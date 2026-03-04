import http from "k6/http";
import { check, sleep } from "k6";
import { uuidv4 } from "https://jslib.k6.io/k6-utils/1.4.0/index.js";

export const options = {
    stages: [
        { duration: "30s", target: 50 },
        { duration: "1m", target: 200 },
        { duration: "2m", target: 350 },
        { duration: "2m", target: 500 },
        { duration: "1m", target: 200 },
        { duration: "30s", target: 0 },
    ],

    thresholds: {
        http_req_duration: ["p(95)<300"],
        http_req_failed: ["rate<0.01"],
    },
};

export default function () {
    const url = "http://localhost:8080/signup";

    const uniqueEmail = `user_${uuidv4()}@mail.com`;

    const payload = JSON.stringify({
        name: "User Testing",
        email: uniqueEmail,
        password: "rahasia",
        password_confirmation: "rahasia",
    });

    const params = {
        headers: {
            "Content-Type": "application/json",
        },
    };

    const res = http.post(url, payload, params);

    check(res, {
        "status is 201": (r) => r.status === 201,
    });

    if (res.status !== 201) {
        console.log(`Error: ${res.status}`);
    }

    sleep(1);
}