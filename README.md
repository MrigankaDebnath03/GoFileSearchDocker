# Product Search API

A high-performance product search microservice built with Go, featuring full-text search capabilities, caching, and PostgreSQL integration.

## Features

- **Full-text Search**: Powered by Bleve search engine for fast and accurate product searches
- **Caching**: LRU cache implementation for improved read performance
- **Concurrent Processing**: Utilizes Go's concurrency features for handling requests
- **PostgreSQL Integration**: Reliable data storage and retrieval
- **Containerized**: Docker and Docker Compose for easy deployment
- **Graceful Shutdown**: Proper server shutdown handling

## Tech Stack

- **Go 1.24**: Modern, concurrent programming language
- **Chi Router**: Lightweight and fast HTTP routing
- **Bleve**: Full-text search and indexing library
- **PostgreSQL**: Relational database for data persistence
- **LRU Cache**: In-memory caching for frequently accessed data
- **Docker & Docker Compose**: Containerization and orchestration

## System Architecture

The application follows a simple yet effective architecture:

1. **API Layer**: Chi router handles HTTP requests
2. **Search Engine**: Bleve provides in-memory full-text search capabilities
3. **Cache Layer**: LRU cache for fast data retrieval
4. **Database Layer**: PostgreSQL for persistent storage

## Getting Started

### Prerequisites

- Docker and Docker Compose
- Git
- Postman (for testing)

### Clone the Repository

```bash
git clone https://github.com/yourusername/product-search-api.git
cd product-search-api
```

### Run with Docker Compose

1. Build and start the containers:

```bash
docker-compose up -d
```

This command will:
- Build the Go application
- Start a PostgreSQL container
- Initialize the database
- Start the API server on port 8080

2. Check if containers are running:

```bash
docker-compose ps
```

### Environment Variables

The application uses the following environment variables (already set in docker-compose.yml):

| Variable | Description | Default |
|----------|-------------|---------|
| DB_HOST | PostgreSQL host | postgres |
| DB_PORT | PostgreSQL port | 5432 |
| DB_USER | Database username | productuser |
| DB_PASSWORD | Database password | productpass |
| DB_NAME | Database name | products |
| CACHE_SIZE | LRU cache size | 10000 |

## API Endpoints

### Search Products

```
GET /search?q={query}
```

Search for products by name.

**Parameters:**
- `q` (required): Search query string

**Response:** JSON array of products matching the search query

### Add Product

```
POST /products
```

Add a new product.

**Request Body:**
```json
{
  "name": "Product Name",
  "category": "Category Name"
}
```

**Response:** JSON object of the created product including the assigned ID

### Delete Product

```
DELETE /products/{id}
```

Delete a product by ID.

**Parameters:**
- `id` (required): Product ID

**Response:** Empty response with 204 No Content status

## Testing with Postman

### Import Postman Collection

You can import the Postman collection provided below to quickly test the API.

### Postman Examples

#### Add a Product

- **Method**: POST
- **URL**: `http://localhost:8080/products`
- **Headers**: 
  - Content-Type: application/json
- **Body**:
```json
{
  "name": "iPhone 15 Pro",
  "category": "Electronics"
}
```

**Example Response**:
```json
{
  "id": 1,
  "name": "iPhone 15 Pro",
  "category": "Electronics"
}
```

#### Add More Products for Testing

- **Method**: POST
- **URL**: `http://localhost:8080/products`
- **Headers**: 
  - Content-Type: application/json
- **Body**:
```json
{
  "name": "Samsung Galaxy S24",
  "category": "Electronics"
}
```

Repeat with different products to build your test dataset:
```json
{
  "name": "MacBook Pro 16-inch",
  "category": "Computers"
}
```

```json
{
  "name": "AirPods Pro",
  "category": "Audio"
}
```

#### Search for Products

- **Method**: GET
- **URL**: `http://localhost:8080/search?q=pro`

This will search for all products containing "pro" in their name.

**Example Response**:
```json
[
  {
    "id": 1,
    "name": "iPhone 15 Pro",
    "category": "Electronics"
  },
  {
    "id": 3,
    "name": "MacBook Pro 16-inch",
    "category": "Computers"
  },
  {
    "id": 4,
    "name": "AirPods Pro",
    "category": "Audio"
  }
]
```

#### Delete a Product

- **Method**: DELETE
- **URL**: `http://localhost:8080/products/1`

This will delete the product with ID 1.

**Example Response**: Empty response with 204 No Content status

## Performance Considerations

The application is designed for performance:

1. **In-memory Search Index**: Bleve index for fast search operations
2. **LRU Cache**: Reduces database load for frequently accessed products
3. **Concurrent Processing**: Search results are processed concurrently
4. **Connection Pooling**: PostgreSQL connection pool for efficient database access

## Shutting Down

To stop and remove the containers:

```bash
docker-compose down
```

To stop the containers but keep the volumes:

```bash
docker-compose stop
```

## Development Notes

### Database Schema

The application uses a simple product schema:

```sql
CREATE TABLE IF NOT EXISTS products (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  category TEXT NOT NULL
)
```

### Search Indexing

The application indexes the product name for full-text search using Bleve's English analyzer.

## Troubleshooting

### Database Connection Issues

If the application fails to connect to the database, ensure PostgreSQL is running:

```bash
docker-compose ps postgres
```

The `wait-for.sh` script is included to ensure the application only starts after the database is ready.

### Search Not Working

If searches return unexpected results, try restarting the application to rebuild the search index:

```bash
docker-compose restart app
```

## License

[MIT License](LICENSE)
