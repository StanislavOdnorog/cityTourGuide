import type { AxiosInstance, Method } from 'axios';
import createClient from 'openapi-fetch';
import type { paths } from './schema';

const OPENAPI_BASE_URL = 'http://openapi.local/api/v1';

function createAxiosFetch(client: AxiosInstance): typeof fetch {
  return async (input, init) => {
    const request = new Request(input, init);
    const url = new URL(request.url);
    const contentType = request.headers.get('content-type') ?? '';

    let data: unknown;
    if (request.method !== 'GET' && request.method !== 'HEAD') {
      if (contentType.includes('application/json')) {
        const text = await request.text();
        data = text ? JSON.parse(text) : undefined;
      } else if (contentType.includes('multipart/form-data')) {
        data = await request.formData();
      } else {
        const text = await request.text();
        data = text || undefined;
      }
    }

    const response = await client.request({
      url: `${url.pathname}${url.search}`,
      method: request.method as Method,
      headers: Object.fromEntries(request.headers.entries()),
      data,
      validateStatus: () => true,
    });

    const headers = new Headers();
    for (const [key, value] of Object.entries(response.headers)) {
      if (Array.isArray(value)) {
        headers.set(key, value.join(', '));
      } else if (value !== undefined) {
        headers.set(key, String(value));
      }
    }

    const body =
      response.data == null || typeof response.data === 'string'
        ? response.data
        : JSON.stringify(response.data);

    return new Response(body, {
      status: response.status,
      headers,
    });
  };
}

export function createGeneratedApiClient(client: AxiosInstance) {
  return createClient<paths>({
    baseUrl: OPENAPI_BASE_URL,
    fetch: createAxiosFetch(client),
  });
}
