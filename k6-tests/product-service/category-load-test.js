import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
    stages: [
        { duration: "20s", target: 20 },
        { duration: "40s", target: 50 },
        { duration: "40s", target: 50 },
        { duration: "20s", target: 0 },
    ],

    thresholds: {
        http_req_duration: ["p(95)<100"],
        http_req_failed: ["rate<0.01"],
    },
};

export default function () {
    const url = "http://localhost:8081/categories";

    const params = {
        headers: {
            "Content-Type": "application/json",
        },
    };

    const res = http.get(url, params);

    check(res, {
        "status is 200": (r) => r.status === 200,
    });

    sleep(1);
}