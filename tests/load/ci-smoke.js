import http from 'k6/http'
import { check, sleep } from 'k6'

export const options = {
  vus: 5,
  duration: '20s',
  thresholds: {
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<800'],
  },
}

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080'

export default function () {
  const health = http.get(`${BASE_URL}/health`)
  check(health, {
    'health is 200': (r) => r.status === 200,
  })

  const ready = http.get(`${BASE_URL}/ready`)
  check(ready, {
    'ready is 200': (r) => r.status === 200,
  })

  const unauthorizedRooms = http.get(`${BASE_URL}/api/v1/rooms`)
  check(unauthorizedRooms, {
    'rooms unauthorized is 401': (r) => r.status === 401,
  })

  sleep(1)
}
