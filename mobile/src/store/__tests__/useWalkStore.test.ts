import { useWalkStore } from '../useWalkStore';

describe('useWalkStore', () => {
  beforeEach(() => {
    // Reset store between tests
    useWalkStore.setState({
      isWalking: false,
      currentLocation: null,
    });
  });

  it('has correct initial state', () => {
    const state = useWalkStore.getState();
    expect(state.isWalking).toBe(false);
    expect(state.currentLocation).toBeNull();
  });

  it('startWalking sets isWalking to true', () => {
    useWalkStore.getState().startWalking();
    expect(useWalkStore.getState().isWalking).toBe(true);
  });

  it('stopWalking sets isWalking to false and clears location', () => {
    useWalkStore.getState().startWalking();
    useWalkStore.getState().updateLocation({
      lat: 41.7,
      lng: 44.8,
      heading: 90,
      speed: 1.2,
    });
    useWalkStore.getState().stopWalking();

    const state = useWalkStore.getState();
    expect(state.isWalking).toBe(false);
    expect(state.currentLocation).toBeNull();
  });

  it('updateLocation stores the location', () => {
    const loc = { lat: 41.7151, lng: 44.8271, heading: 180, speed: 0.8 };
    useWalkStore.getState().updateLocation(loc);

    const state = useWalkStore.getState();
    expect(state.currentLocation).toEqual(loc);
  });

  it('updateLocation replaces previous location', () => {
    useWalkStore.getState().updateLocation({
      lat: 41.7,
      lng: 44.8,
      heading: 90,
      speed: 1.0,
    });
    const newLoc = { lat: 41.72, lng: 44.83, heading: 180, speed: 1.5 };
    useWalkStore.getState().updateLocation(newLoc);

    expect(useWalkStore.getState().currentLocation).toEqual(newLoc);
  });
});
