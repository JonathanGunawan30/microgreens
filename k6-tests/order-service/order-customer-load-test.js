import http from "k6/http";
import { check, sleep } from "k6";

const USER_BASE_URL = "http://localhost:8080";
const ORDER_BASE_URL = "http://localhost:8082";

export const options = {
    stages: [
        { duration: "10s", target: 10 },
        { duration: "20s", target: 20 },
        { duration: "20s", target: 20 },
        { duration: "10s", target: 0 },
    ],

    thresholds: {
        http_req_duration: ["p(95)<200"],
        http_req_failed: ["rate<0.01"],
    },
};

export function setup() {
    const loginPayload = JSON.stringify({
        email: "jonathan1@mail.com",
        password: "rahasia",
    });

    const loginRes = http.post(`${USER_BASE_URL}/signin`, loginPayload, {
        headers: { "Content-Type": "application/json" },
    });

    check(loginRes, {
        "login success": (r) => r.status === 200,
    });

    const token = loginRes.json("data.access_token");

    return { token };
}

export default function (data) {

    const params = {
        headers: {
            Authorization: `Bearer ${data.token}`,
            "Content-Type": "application/json",
        },
    };

    sleep(1);

    const resOrders = http.get(`${ORDER_BASE_URL}/auth/orders`, params);

    check(resOrders, {
        "orders status 200": (r) => r.status === 200,
    });

    sleep(1);

    const orderIds = [14, 15];
    const orderId = orderIds[Math.floor(Math.random() * orderIds.length)];

    const resOrderDetail = http.get(
        `${ORDER_BASE_URL}/auth/orders/${orderId}`,
        params
    );

    check(resOrderDetail, {
        "order detail success": (r) => r.status === 200 || r.status === 404,
    });

    sleep(1);
}