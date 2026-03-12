import { useCallback, useEffect, useRef } from 'react';
import { fetchCityDownloadManifest } from '@/api/endpoints';
import { normalizeError, userMessageForError } from '@/api/errors';
import { StoryCacheManager } from '@/services/cache';
import type { CancelableDownload } from '@/services/cache/StoryCacheManager';
import { useAuthStore } from '@/store/useAuthStore';
import { useDownloadStore } from '@/store/useDownloadStore';
import type { DownloadManifestItem, NearbyStoryCandidate } from '@/types';

/**
 * Convert a DownloadManifestItem to a NearbyStoryCandidate shape
 * so it can be passed to StoryCacheManager.
 */
function toCandidate(item: DownloadManifestItem): NearbyStoryCandidate {
  return {
    poi_id: item.poi_id,
    poi_name: item.poi_name,
    poi_lat: 0,
    poi_lng: 0,
    story_id: item.story_id,
    story_text: '',
    audio_url: item.audio_url,
    duration_sec: item.duration_sec,
    distance_m: 0,
    score: 0,
  };
}

/**
 * Hook for downloading all stories for a city for offline use.
 * Provides byte-level progress, cancellation safety, and unmount cleanup.
 */
export function useDownloadCity(cityId: number, language = 'en') {
  const cityDownload = useDownloadStore(
    useCallback((state) => state.downloadsByCityId[String(cityId)], [cityId]),
  );
  const downloadedCities = useDownloadStore((state) => state.downloadedCities);
  const setCityStatus = useDownloadStore((state) => state.setCityStatus);
  const setCityProgress = useDownloadStore((state) => state.setCityProgress);
  const setCityError = useDownloadStore((state) => state.setCityError);
  const markCityDownloaded = useDownloadStore((state) => state.markCityDownloaded);
  const resetCityDownload = useDownloadStore((state) => state.resetCityDownload);

  const cancelledRef = useRef(false);
  const mountedRef = useRef(true);
  const activeDownloadRef = useRef<CancelableDownload | null>(null);
  const cacheManagerRef = useRef<StoryCacheManager | null>(null);

  const isDownloaded = String(cityId) in downloadedCities;
  const status = cityDownload?.status ?? 'idle';
  const progress = cityDownload?.progress ?? {
    completedBytes: 0,
    totalBytes: 0,
    completedFiles: 0,
    totalFiles: 0,
  };
  const error = cityDownload?.error ?? null;

  // Cleanup on unmount: prevent stale state updates and cancel in-flight downloads.
  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      cancelledRef.current = true;
      if (activeDownloadRef.current) {
        void activeDownloadRef.current.cancel();
        activeDownloadRef.current = null;
      }
    };
  }, []);

  // If the hook instance is reused for a different city, cancel the old city's work.
  useEffect(() => {
    cancelledRef.current = false;
    return () => {
      cancelledRef.current = true;
      if (activeDownloadRef.current) {
        void activeDownloadRef.current.cancel();
        activeDownloadRef.current = null;
      }
    };
  }, [cityId]);

  const startDownload = useCallback(async () => {
    // Guard: don't duplicate a download for an already-downloaded city
    if (useDownloadStore.getState().isCityDownloaded(cityId)) {
      return;
    }

    cancelledRef.current = false;
    resetCityDownload(cityId);
    setCityStatus(cityId, 'fetching_manifest');

    try {
      const manifestLanguage = language === 'ru' ? 'ru' : 'en';
      const manifest = await fetchCityDownloadManifest(cityId, manifestLanguage);

      if (cancelledRef.current) return;

      const items = manifest.data.filter((item) => item.audio_url !== null);
      if (items.length === 0) {
        if (mountedRef.current) {
          setCityStatus(cityId, 'completed');
          markCityDownloaded({
            cityId,
            language,
            downloadedAt: Date.now(),
            storyIds: [],
            totalFiles: 0,
            totalSizeBytes: 0,
          });
        }
        return;
      }

      if (mountedRef.current) {
        setCityStatus(cityId, 'downloading');
        setCityProgress(cityId, {
          totalBytes: manifest.total_size_bytes,
          totalFiles: items.length,
          completedBytes: 0,
          completedFiles: 0,
        });
      }

      // Initialize cache manager
      if (!cacheManagerRef.current) {
        cacheManagerRef.current = new StoryCacheManager();
        await cacheManagerRef.current.init();
      }
      const cacheManager = cacheManagerRef.current;

      let completedBytesBase = 0;
      let completedFiles = 0;

      for (const item of items) {
        if (cancelledRef.current) {
          if (mountedRef.current) {
            resetCityDownload(cityId);
          }
          return;
        }

        const candidate = toCandidate(item);
        const fileSizeEstimate = item.file_size_bytes || 0;

        const handle = cacheManager.downloadAudioWithProgress(candidate, (progressEvent) => {
          if (cancelledRef.current || !mountedRef.current) return;
          const currentFileBytes =
            fileSizeEstimate > 0
              ? Math.min(progressEvent.bytesWritten, fileSizeEstimate)
              : progressEvent.bytesWritten;
          setCityProgress(cityId, {
            completedBytes: completedBytesBase + currentFileBytes,
          });
        });

        activeDownloadRef.current = handle;
        const result = await handle.promise;
        activeDownloadRef.current = null;

        if (cancelledRef.current) {
          if (mountedRef.current) {
            resetCityDownload(cityId);
          }
          return;
        }

        if (result !== null) {
          completedFiles++;
          completedBytesBase += fileSizeEstimate;
          if (mountedRef.current) {
            setCityProgress(cityId, {
              completedFiles,
              completedBytes: completedBytesBase,
            });
          }
        }
      }

      if (!cancelledRef.current && mountedRef.current) {
        setCityStatus(cityId, 'completed');
        markCityDownloaded({
          cityId,
          language,
          downloadedAt: Date.now(),
          storyIds: items.map((i) => i.story_id),
          totalFiles: items.length,
          totalSizeBytes: manifest.total_size_bytes,
        });
      }
    } catch (err) {
      if (!cancelledRef.current && mountedRef.current) {
        const appErr = normalizeError(err);

        if (appErr.category === 'unauthorized') {
          useAuthStore.getState().clearSession();
        }

        setCityStatus(cityId, 'error');
        setCityProgress(cityId, {
          completedBytes: 0,
          completedFiles: 0,
          totalBytes: 0,
          totalFiles: 0,
        });
        setCityError(cityId, userMessageForError(appErr));
      }
    }
  }, [
    cityId,
    language,
    resetCityDownload,
    setCityStatus,
    setCityProgress,
    setCityError,
    markCityDownloaded,
  ]);

  const cancelDownload = useCallback(() => {
    cancelledRef.current = true;
    if (activeDownloadRef.current) {
      void activeDownloadRef.current.cancel();
      activeDownloadRef.current = null;
    }
    resetCityDownload(cityId);
  }, [cityId, resetCityDownload]);

  return {
    status,
    progress,
    error,
    isDownloaded,
    startDownload,
    cancelDownload,
  };
}
