FROM node:20-alpine AS builder

# Create app directory
WORKDIR /app

# Copy package files and .npmrc
COPY package*.json ./.npmrc ./ .env ./

# Install dependencies
RUN npm ci

# Copy source
COPY . .

# Build the app
RUN npm run build \
    && rm -f .npmrc  # Remove .npmrc after build

FROM node:20-alpine
WORKDIR /app
RUN npm install -g serve

ENV NODE_ENV=production
ENV VITE_API_URL=https://trenova.local/api/v1

COPY --from=builder /app/dist ./dist

EXPOSE 5173
CMD ["serve", "-s", "dist", "-l", "5173"]