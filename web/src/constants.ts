export const MAP_STYLE = 'https://tiles.openfreemap.org/styles/liberty';

// Location Service — API Gateway URL (proxied to location-service:8084)
export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Default simulation user
export const SIM_USER_ID = 'user-sim-001';

// Simulation interval (ms) between auto-movement steps
export const SIM_INTERVAL_MS = 2500;
