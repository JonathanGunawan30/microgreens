import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
    stages: [
        { duration: "30s", target: 20 },
        { duration: "1m", target: 50 },
        { duration: "1m", target: 50 },
        { duration: "30s", target: 0 },
    ],

    thresholds: {
        http_req_duration: ["p(95)<150"],
        http_req_failed: ["rate<0.01"],
    },
};

const BASE_URL = "http://localhost:8081";

export default function () {

    let resFeaturedProducts = http.get(`${BASE_URL}/products/featured`);

    check(resFeaturedProducts, {
        "featured products 200": (r) => r.status === 200,
    });

    sleep(1);

    let resCategories = http.get(`${BASE_URL}/categories`);

    check(resCategories, {
        "categories 200": (r) => r.status === 200,
    });

    sleep(1);

    let resProducts = http.get(`${BASE_URL}/products`);

    check(resProducts, {
        "products 200": (r) => r.status === 200,
    });

    sleep(1);

    const products = resProducts.json();

    if (products.length > 0) {
        const randomIndex = Math.floor(Math.random() * products.length);
        const productId = products[randomIndex].id;

        const resProductDetail = http.get(`${BASE_URL}/products/${productId}`);

        check(resProductDetail, {
            "product detail 200": (r) => r.status === 200,
        });
    }

    sleep(2);

    http.get(`${BASE_URL}/categories`);

    sleep(1);
}