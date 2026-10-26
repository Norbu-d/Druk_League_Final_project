import http from 'k6/http';
import { check, sleep } from 'k6';

const baseURL = __ENV.API_BASE_URL || 'http://127.0.0.1:8080';
const username = __ENV.BOOTSTRAP_ADMIN_USER || 'ci-admin';
const password = __ENV.BOOTSTRAP_ADMIN_PASSWORD || 'ci-only-admin-password';
const tournamentID = __ENV.TOURNAMENT_ID || '00000000-0000-4000-8000-000000000000';

export const options = {
  scenarios: {
    login: {
      executor: 'constant-vus',
      exec: 'login',
      vus: 2,
      duration: '30s',
    },
    publicStandings: {
      executor: 'constant-vus',
      exec: 'publicStandings',
      vus: 5,
      duration: '30s',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    'http_req_duration{operation:login}': ['p(95)<1000'],
    'http_req_duration{operation:public-standings}': ['p(95)<500'],
  },
};

export function login() {
  const response = http.post(
    `${baseURL}/api/auth/login`,
    JSON.stringify({ username, password }),
    {
      headers: { 'Content-Type': 'application/json' },
      tags: { operation: 'login' },
    },
  );

  check(response, {
    'login returns 200': (res) => res.status === 200,
    'login returns a token': (res) => Boolean(res.json('token')),
  });
  sleep(0.5);
}

export function publicStandings() {
  const response = http.get(
    `${baseURL}/api/standings?tournament_id=${encodeURIComponent(tournamentID)}`,
    { tags: { operation: 'public-standings' } },
  );

  check(response, {
    'public standings returns 200 without a token': (res) => res.status === 200,
    'public standings returns a JSON array': (res) => {
      try {
        return Array.isArray(res.json());
      } catch {
        return false;
      }
    },
  });
  sleep(0.5);
}
