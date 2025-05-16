# Farmako Assignment Coupon API

This project implements a high-performance Coupon API for Farmako, designed using Go and PostgreSQL, with Docker-based deployment. The service provides endpoints to create, update, and validate discount coupons and efficiently fetch applicable coupons for user carts.

---

## 🧱 Architecture

- **Language**: Go (Golang)
- **Database**: PostgreSQL
- **Containerization**: Docker
- **Caching**: In-memory caching with TTL
- **Concurrency Control**: Row-level locking for consistency

---

## 📌 API Endpoints

### 1. **Add Coupons**

- **Endpoint**: `POST /admin/addCoupons`
- **Description**: Allows an admin to add new coupon definitions.
- **Body**: Coupon details including applicable medicines/categories, limits, and discount info.

### 2. **Update Coupon**

- **Endpoint**: `PUT /coupon/update`
- **Description**: Update an existing coupon’s details.
- **Body**: Coupon ID and fields to update.

### 3. **Get Applicable Coupons**

- **Endpoint**: `POST /coupon/applicable`
- **Description**: Returns all coupons applicable to a user's cart based on the medicines and categories in the cart.
- **Body**: List of cart items (medicine IDs and quantities).

### 4. **Validate Coupon**

- **Endpoint**: `POST /coupon/validate`
- **Description**: Validates if a given coupon is applicable for the cart and calculates the discount if valid.
- **Body**: Coupon code and cart items.

---

## 🚀 Caching Strategy

- **TTL Caching on Medicine Lookup**:  
  Frequently queried medicine data during the `/coupon/applicable` call is cached with a time-to-live (TTL) strategy.  
  This reduces repeated database hits for common items and improves overall performance.

---

## 🔒 Concurrency Strategy

- **Row-Level Locking During Coupon Validation**:  
  To ensure correctness in multi-user scenarios, the system locks the row corresponding to the coupon usage when validating and updating the usage count.  
  This avoids race conditions and ensures data consistency when multiple users try to redeem the same coupon simultaneously.

---

## 🐳 Running the Project

Make sure you have Docker and Docker Compose installed.

1. Clone the repo.
2. Run:

   ```bash
   docker-compose up --build
   ```
