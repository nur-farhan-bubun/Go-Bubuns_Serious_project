export const MAP_STYLE = 'https://tiles.openfreemap.org/styles/liberty';

// Location Service — API Gateway URL (proxied to location-service:8084)
export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Default simulation user — overridden by actual logged-in user at runtime
export const SIM_USER_ID = '';

// Simulation interval (ms) between auto-movement steps
export const SIM_INTERVAL_MS = 2500;
