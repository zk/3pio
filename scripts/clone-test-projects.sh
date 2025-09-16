#!/bin/bash

# Clone all open source projects for 3pio testing
# This script clones projects in parallel for faster execution

cd open-source || exit 1

echo "======================================"
echo "Cloning Open Source Projects for 3pio"
echo "======================================"

# Jest projects
echo ""
echo "📦 Cloning Jest projects..."
git clone --depth 1 https://github.com/facebook/create-react-app.git &
git clone --depth 1 https://github.com/reduxjs/redux-toolkit.git &
git clone --depth 1 https://github.com/remix-run/react-router.git &
git clone --depth 1 https://github.com/axios/axios.git &
git clone --depth 1 https://github.com/jestjs/jest.git &
git clone --depth 1 https://github.com/mui/material-ui.git &
wait
echo "✅ Jest projects cloned"

# Vitest projects
echo ""
echo "📦 Cloning Vitest projects..."
git clone --depth 1 https://github.com/vueuse/vueuse.git &
git clone --depth 1 https://github.com/unocss/unocss.git &
git clone --depth 1 https://github.com/vuejs/pinia.git &
git clone --depth 1 https://github.com/unjs/unplugin.git &
git clone --depth 1 https://github.com/nuxt/nuxt.git &
git clone --depth 1 https://github.com/vitest-dev/vitest.git &
wait
echo "✅ Vitest projects cloned"

# Go projects
echo ""
echo "📦 Cloning Go projects..."
git clone --depth 1 https://github.com/go-yaml/yaml.git go-yaml &
git clone --depth 1 https://github.com/google/uuid.git &
git clone --depth 1 https://github.com/gin-gonic/gin.git &
git clone --depth 1 https://github.com/labstack/echo.git &
git clone --depth 1 https://github.com/gofiber/fiber.git &
git clone --depth 1 https://github.com/kubernetes/kubernetes.git &
git clone --depth 1 https://github.com/docker/cli.git docker-cli &
git clone --depth 1 https://github.com/etcd-io/etcd.git &
git clone --depth 1 https://github.com/prometheus/prometheus.git &
wait
echo "✅ Go projects cloned"

# Python projects
echo ""
echo "📦 Cloning Python projects..."
git clone --depth 1 https://github.com/httpie/httpie.git &
git clone --depth 1 https://github.com/pallets/click.git &
git clone --depth 1 https://github.com/pallets/flask.git &
git clone --depth 1 https://github.com/psf/requests.git &
git clone --depth 1 https://github.com/pandas-dev/pandas.git &
git clone --depth 1 https://github.com/scikit-learn/scikit-learn.git &
git clone --depth 1 https://github.com/django/django.git &
wait
echo "✅ Python projects cloned"

# Rust projects
echo ""
echo "📦 Cloning Rust projects..."
git clone --depth 1 https://github.com/serde-rs/serde.git &
git clone --depth 1 https://github.com/clap-rs/clap.git &
git clone --depth 1 https://github.com/actix/actix-web.git &
git clone --depth 1 https://github.com/tokio-rs/tokio.git &
git clone --depth 1 https://github.com/rust-lang/rust.git &
git clone --depth 1 https://github.com/denoland/deno.git &
wait
echo "✅ Rust projects cloned"

echo ""
echo "======================================"
echo "✨ All projects cloned successfully!"
echo "======================================"
echo ""
echo "Total projects cloned: 34"
echo ""
echo "You can now test 3pio with these projects:"
echo "  Jest:    6 projects"
echo "  Vitest:  6 projects"
echo "  Go:      9 projects"
echo "  Python:  7 projects"
echo "  Rust:    6 projects"