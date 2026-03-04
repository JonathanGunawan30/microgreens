import http from 'k6/http'
import {check, sleep} from 'k6';
import {uuidv4} from 'https://jslib.k6.io/k6-utils/1.4.0/index.js'

export const options = {
    vus: 30,
    duration: '30s',
};

export default function() {
    const url = 'http://localhost:8080/signup';

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

    const res = http.post(url, payload, params)

    check(res, {
        "status is 201": (r) => r.status === 201,
        "response say success": (r) => r.json("message") === "Success",
    });

    sleep(1);
}