import { defineConfig } from "vitest/config";

export default defineConfig({
    test: {
        globals: false,
        environment: "node",
        include: ["src/**/*.test.ts"],
    },
    resolve: {
        alias: {
            // Allow .js imports to resolve to .ts files
            "^(.+)\\.js$": "$1.ts",
        },
    },
    esbuild: {
        target: "es2022",
    },
});
