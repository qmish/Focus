import http from 'k6/http'
import { check, sleep } from 'k6'

const API_URL = __ENV.API_URL || 'https://api-stage.company.com'
const JITSI_URL = __ENV.JITSI_URL || 'https://meet-stage.company.com'
const JVB_HEALTH_URL = __ENV.JVB_HEALTH_URL || 'https://jvb-stage.company.com/about/health'

export const options = {
  scenarios: {
    api_read_profile: {
      executor: 'ramping-vus',
      startVUs: 5,
      stages: [
        { duration: '1m', target: 20 },
        { duration: '2m', target: 50 },
        { duration: '1m', target: 0 },
      ],
      exec: 'apiReadProfile',
    },
    jitsi_health_profile: {
      executor: 'constant-vus',
      vus: 15,
      duration: '3m',
      exec: 'jitsiHealthProfile',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.03'],
    http_req_duration: ['p(95)<1200'],
  },
}

export function apiReadProfile() {
  const health = http.get(`${API_URL}/health`)
  check(health, { 'api /health is 200': (r) => r.status === 200 })

  const ready = http.get(`${API_URL}/ready`)
  check(ready, { 'api /ready is 200': (r) => r.status === 200 })

  sleep(1)
}

export function jitsiHealthProfile() {
  const webHealth = http.get(`${JITSI_URL}/`)
  check(webHealth, { 'jitsi web is reachable': (r) => r.status >= 200 && r.status < 500 })

  const jvbHealth = http.get(JVB_HEALTH_URL)
  check(jvbHealth, { 'jvb health is 200': (r) => r.status === 200 })

  sleep(1)
}
