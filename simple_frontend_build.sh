#!/bin/bash

echo "Building React frontend..."
cd frontend
npm install
npm run build

echo "Copying frontend build to web/static/dist..."
mkdir -p ../web/static/dist
mkdir -p ../web/static/dist/assets

if [ -d "build" ]; then
  cp -r build/* ../web/static/dist/
elif [ -d "dist" ]; then
  cp -r dist/* ../web/static/dist/
else
  echo "Creating basic fallback index.html"
  echo "<!DOCTYPE html><html><head><title>Claude Squad</title></head><body><h1>Claude Squad</h1></body></html>" > ../web/static/dist/index.html
fi

echo "Frontend build completed"