# go-htmx-blog

# steps to run this site

1. go mod tidy
2. go run main.go
3. (OPTIONAL) To setup hot reloading, give execute permission to the entr-reload.sh script, and run it instead of go run main.go

## To run tailwindCSS with some custom presets

1. Follow steps [here](https://tailwindcss.com/blog/standalone-cli) to download tailwind executable
2. After initializing, run this command in a new tab (which is based on the file paths in this repo) to develop locally

```bash
 ./tailwindcss -i input.css -o ./static/output.css --watch
```

3. To prepare for production,

```bash
./tailwindcss -i input.css -o ./static/output.css --minify
```
