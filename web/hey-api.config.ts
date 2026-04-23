import { defineConfig } from "@hey-api/openapi-ts";

export default defineConfig({
	input: "../api/openapi.yaml",
	output: {
		path: "src/api/generated",
	},
	plugins: [
		"@hey-api/client-fetch",
		"@hey-api/sdk",
		"@hey-api/typescript",
		"@tanstack/react-query",
		"@hey-api/schemas",
	],
});
