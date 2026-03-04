import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
    scenarios: {
        products_list: {
            executor: "ramping-vus",
            startVUs: 0,
            stages: [
                { duration: "20s", target: 20 },
                { duration: "40s", target: 50 },
                { duration: "40s", target: 50 },
                { duration: "20s", target: 0 },
            ],
            exec: "productsList",
        },

        products_featured: {
            executor: "ramping-vus",
            startVUs: 0,
            stages: [
                { duration: "20s", target: 10 },
                { duration: "40s", target: 20 },
                { duration: "40s", target: 20 },
                { duration: "20s", target: 0 },
            ],
            exec: "productsFeatured",
        },
    },

    thresholds: {
        http_req_duration: ["p(95)<150"],
        http_req_failed: ["rate<0.01"],
    },
};

export function productsList() {
    const res = http.get("http://localhost:8081/products");

    check(res, {
        "products status 200": (r) => r.status === 200,
    });

    sleep(1);
}

export function productsFeatured() {
    const res = http.get("http://localhost:8081/products/featured");

    check(res, {
        "featured status 200": (r) => r.status === 200,
    });

    sleep(1);
}