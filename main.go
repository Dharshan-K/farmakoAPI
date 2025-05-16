package main

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"context"
	"log"
	"time"
	"sync"
	"github.com/google/uuid"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/swagger"
	"github.com/dgraph-io/ristretto"
	_ "github.com/Dharshan-K/farmakoAPI/docs"
)

//Medicine represents the Medicine data
type Medicine struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Category string    `json:"category"`
	Price    float64   `json:"price"`
}

//OrderInput represents the Order Input given to a API
type OrderInput struct {
	CartItems  []Medicine `json:"cart_items"`
	OrderTotal float64   `json:"order_total"`
	Timestamp  time.Time `json:"timestamp"`
}

// ApplicableCoupon represents the result of query of eligible coupons
type ApplicableCoupon struct {
	CouponCode    string  `json:"coupon_code"`
	DiscountValue float64 `json:"discount_value"`
}

// ValidateCoupon is used in /coupon/validate
type ValidateCoupon struct {
	UserID uuid.UUID `json:"user_id"`
	CouponCode string `json:"coupon_code"`
	OrderInput
}
//UpdateCouponRequest is for update info of a coupon
type UpdateCouponRequest struct {
	CouponCode string `json:"coupon_code"`
	DiscountType string `json:"discount_type"`
	DiscountValue float64 `json:"discount_value"`
	MaxUsagePerUser int `json:"max_usage_per_user"`
}

//CouponUsageCache is a cache to track coupon usage
type CouponUsageCache struct {
	cache *ristretto.Cache
	mu    sync.Mutex
}

//Coupon handles and validates the coupon data
type CouponData struct {
	CouponCode string `json:"coupon_code" validate:"required,min=3,max=50"`
	ExpiryDate time.Time `json:"expiry_date" validate:"required"`
	ApplicableMedicineId []string `json:"applicable_medicine_id" validate="dive,uuid4"`
	ApplicableCategories []string `json:"applicable_categories" validate="dive,required"`
	UsageType string `json:"usage_type" validate:"required,oneof=one_time multi_use time_based"`
	MinOrderValue float64 `json:"min_order_value" validate:"gte=0"`
	ValidFrom time.Time `json:"valid_from" validate:"required"`
	ValidUntil time.Time `json:"valid_until" validate:"required,gtfield=ValidFrom"`
	TermsAndConditions string `json:"terms_and_conditions"`
	DiscountType string `json:"discount_type" validate:"required,oneof=flat percentage"`
	DiscountValue float64 `json:"discount_value" validate:"gt=0,lt=100"`
	DiscountTarget string `json:"discount_Target" validate:"required,oneof=inventory charges inventory_and_charges"`
	MaxUsagePerUser int `json:"max_usage_per_user" validate:"gt=0"`
}

// AddCoupon godoc
// @Summary Add a new coupon
// @Description Add a new coupon to the system
// @Tags Admin
// @Accept json
// @Produce json
// @Param coupon body CouponData true "Coupon data"
// @Success 200 {string} string "Coupon added successfully"
// @Failure 400 {object} map[string]interface{} "Validation errors"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /admin/addCoupons [post]
func addCouponHandler(c *fiber.Ctx, connPool *pgxpool.Pool) error {
	var couponData CouponData;
	var validate = validator.New()

	if err := c.BodyParser(&couponData); err != nil {
		fmt.Println("Invalid request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error" : err,
		})
	}

	if err:= validate.Struct(couponData); err != nil {
		errors := make(map[string]string)
		for _,err := range err.(validator.ValidationErrors) {
			errors[err.Field()] = fmt.Sprintf("failed on '%s' validation.", err.Tag())
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"validation_errors" : errors,
		})
	}

	ctx := c.Context()
	tx, err := connPool.Begin(ctx)
	if err !=nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not start transaction"})
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO coupon(coupon_code,
		expiry_date,
		usage_type,
		min_order_value,
		valid_from,
		valid_until,
		discount_type,
		discount_value,
		max_usage_per_user,
		terms_and_conditions,
		discount_target
	)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, couponData.CouponCode, couponData.ExpiryDate, couponData.UsageType, couponData.MinOrderValue, couponData.ValidFrom, couponData.ValidUntil, couponData.DiscountType, couponData.DiscountValue, couponData.MaxUsagePerUser, couponData.TermsAndConditions, couponData.DiscountTarget)
	if err != nil {
		fmt.Println("Error: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error" : err,
		})
	}

	medicineQuery := `INSERT INTO coupon_medicine_map(coupon_code, medicine_id) VALUES($1,$2)`
	categoryQuery := `INSERT INTO coupon_category_map(coupon_code, category_name) VALUES($1,$2)`

	for _,medicineID := range(couponData.ApplicableMedicineId) {
		_, err := tx.Exec(ctx, medicineQuery, couponData.CouponCode,medicineID)
		if err != nil {
			fmt.Println("Error: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error" : err,
			})
		}
	}

	for _, category := range(couponData.ApplicableCategories) {
		_, err := tx.Exec(ctx, categoryQuery, couponData.CouponCode,category)
		if err != nil {
			fmt.Println("Error: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error" : err,
			})
		}
	} 
	if err := tx.Commit(ctx); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to commit transaction"})
	}

	return c.SendString("Coupon added successfully")
}

// UpdateCoupon godoc
// @Summary Update an existing coupon
// @Description Update coupon details
// @Tags Admin
// @Accept json
// @Produce json
// @Param coupon body UpdateCouponRequest true "Coupon update data"
// @Success 200 {object} map[string]interface{} "Success message"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /coupon/update [post]	
func updateCouponHandler(c *fiber.Ctx, connPool *pgxpool.Pool) error {
	var req UpdateCouponRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid input")
	}

	tx, err := connPool.Begin(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer tx.Rollback(c.Context()) 

	var couponID int
	query := `SELECT id FROM coupon WHERE code = $1 FOR UPDATE`
	err = tx.QueryRow(c.Context(), query, req.CouponCode).Scan(&couponID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to lock coupon")
	}

	updateQuery := `
		UPDATE coupons
		SET discount_type = $1,
				discount_value = $2,
				max_usage_per_user = $3
		WHERE id = $4
	`
	_, err = tx.Exec(c.Context(), updateQuery,
		req.DiscountType, req.DiscountValue, req.MaxUsagePerUser, couponID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Update failed")
	}

	if err := tx.Commit(c.Context()); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Transaction commit failed")
	}

	return c.JSON(fiber.Map{
		"message": "Coupon updated successfully",
	})
}

// GetApplicableCoupons godoc
// @Summary Get applicable coupons
// @Description Returns coupons applicable to the provided cart items
// @Tags Coupons
// @Accept json
// @Produce json
// @Param cart body OrderInput true "Cart items"
// @Success 200 {object} map[string][]ApplicableCoupon "List of applicable coupons"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Router /coupon/applicable [post]
func getApplicableCoupons(c *fiber.Ctx, connPool *pgxpool.Pool, cache *ristretto.Cache) error {
	var cart_details OrderInput;
	if err := c.BodyParser(&cart_details); err != nil {
		fmt.Println("Invalid Input")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error" : err,
		})
	}

	medicineIDs := make([]uuid.UUID, len(cart_details.CartItems))
	for i, item := range cart_details.CartItems {
			medicineIDs[i] = item.ID
	}

	var medicines []uuid.UUID;
	var categories []string;
	var missingMedicineIDs []uuid.UUID

	for _, medID := range medicineIDs {
		cacheKey := fmt.Sprintf("medicine:%s", medID.String())

		if val, found := cache.Get(cacheKey); found {
			med := val.(Medicine)
			medicines = append(medicines, med.ID)
			categories = append(categories, med.Category)
		}
	}

	for _, id := range medicineIDs {
		found := false
		for _, cachedID := range medicines {
			if id == cachedID {
				found = true
				break
			}
		}
		if !found {
			missingMedicineIDs = append(missingMedicineIDs, id)
		}
	}

	if len(missingMedicineIDs) > 0 {
		medicine_query := "SELECT id,name,category,price from medicine WHERE id = ANY($1::uuid[])"
		ctx := c.Context()

		rows,err := connPool.Query(ctx, medicine_query, medicineIDs)
		if err != nil {
			fmt.Println("Error retreiving medicine Data")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error" : err,
			})
		}
		defer rows.Close()

		for rows.Next(){
			var m Medicine;
			if err := rows.Scan(&m.ID, &m.Name, &m.Category, &m.Price); err != nil {
				fmt.Println("Error retreiving medicine")
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error" : err,
				})
			}
			cacheKey := fmt.Sprintf("medicine:%s", m.ID.String())
			cache.SetWithTTL(cacheKey, m, 1, time.Hour)

			medicines = append(medicines, m.ID)
			categories = append(categories, m.Category)
		}
	}

	couponQuery := `SELECT DISTINCT c.coupon_code, c.discount_type, c.discount_value, c.min_order_value
	FROM coupon c
	LEFT JOIN coupon_medicine_map cmm ON c.coupon_code = cmm.coupon_code
	LEFT JOIN coupon_category_map ccm ON c.coupon_code = ccm.coupon_code
	WHERE cmm.medicine_id = ANY($1::uuid[]) OR ccm.category_name = ANY($2::text[])
	`

	var applicableCoupons []ApplicableCoupon
	rows,err := connPool.Query(c.Context(), couponQuery, medicines, categories)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error" : err,
		})
	}
	for rows.Next() {
		var coupon_code, discount_type string;
		var discount_value, min_order_value float64;
		if err := rows.Scan(&coupon_code, &discount_type, &discount_value, &min_order_value); err != nil {
			fmt.Println("Error retreiving coupon code")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error" : err,
			})
		}
		var discount float64
		if cart_details.OrderTotal >= min_order_value {
			if discount_type == "flat" {
				discount = discount_value;
			}else if discount_type == "percentage" {
				discount = cart_details.OrderTotal * (discount_value / 100);
			}
			applicableCoupons = append(applicableCoupons, ApplicableCoupon{
				CouponCode : coupon_code,
				DiscountValue : discount,
			})
		}
	}

	return c.JSON(fiber.Map{
		"applicable_coupons": applicableCoupons,
	})
}

// ValidateCoupon godoc
// @Summary Validate a coupon
// @Description Check if a coupon is valid for the given order
// @Tags Coupons
// @Accept json
// @Produce json
// @Param request body ValidateCoupon true "Validation request"
// @Success 200 {object} map[string]interface{} "Validation result"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Router /coupon/validate [post]
func validateCouponHandler(c *fiber.Ctx, connPool *pgxpool.Pool) error {
	var coupon_details ValidateCoupon
	if err := c.BodyParser(&coupon_details); err != nil {
		fmt.Println("Invalid Input")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error" : err,
		})
	}

	fmt.Println("coupon details: %v", coupon_details)

	ctx := c.Context()
	tx, err := connPool.Begin(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to begin transaction"})
	}
	defer tx.Rollback(ctx)

	var currentUsage int
	err = tx.QueryRow(ctx,`SELECT usage FROM coupon_usage WHERE user_id = $1 AND coupon_code = $2 FOR UPDATE`, coupon_details.UserID, coupon_details.CouponCode).Scan(&currentUsage)

	if err != nil {
		if err.Error() == "no rows in result set" {
			currentUsage = 0
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch usage"})
		}
	}

	couponQuery := `SELECT 
		coupon_code,
    expiry_date,
    usage_type,
    min_order_value,
    valid_from,
    valid_until,
    discount_type,
    discount_value,
    discount_target,
    max_usage_per_user,
    terms_and_conditions	
	FROM coupon WHERE coupon_code = $1`
	rows, err := connPool.Query(ctx, couponQuery, coupon_details.CouponCode)
	if err != nil {
		fmt.Println("Error retrieving Coupon details.")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error" : err.Error(),
		})
	}

	var coupon_data CouponData
	for rows.Next(){
		if err = rows.Scan( 
			&coupon_data.CouponCode, 
			&coupon_data.ExpiryDate, 
			&coupon_data.UsageType, 
			&coupon_data.MinOrderValue, 
			&coupon_data.ValidFrom, 
			&coupon_data.ValidUntil, 
			&coupon_data.DiscountType, 
			&coupon_data.DiscountValue, 
			&coupon_data.DiscountTarget,  
			&coupon_data.MaxUsagePerUser,
			&coupon_data.TermsAndConditions); err != nil {
			fmt.Println("Error scanning coupon.")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error" : err.Error(),
			})
		}
	}
	fmt.Println("Coupon data: %v", coupon_data)

	if currentUsage >= coupon_data.MaxUsagePerUser {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"is_valid": false,
			"message":  "Coupon usage limit exceeded for this user",
		})
	}

	if currentUsage == 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO coupon_usage (user_id, coupon_code, usage)
			VALUES ($1, $2, 1)
		`, coupon_details.UserID, coupon_details.CouponCode)
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE coupon_usage
			SET usage = usage + 1
			WHERE user_id = $1 AND coupon_code = $2
		`, coupon_details.UserID, coupon_details.CouponCode)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update usage"})
	}
	
	if err := tx.Commit(ctx); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Transaction commit failed"})
	}

	timestamp := coupon_details.Timestamp
	if coupon_data.ValidFrom.IsZero() && timestamp.Before(coupon_data.ValidFrom) || 
	coupon_data.ValidUntil.IsZero() && timestamp.After(coupon_data.ValidUntil) || 
	timestamp.After(coupon_data.ExpiryDate) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"is_valid": false,
			"reason":   "coupon expired or not applicable",
		})
	}


	if coupon_details.OrderTotal <= coupon_data.MinOrderValue {
		return c.JSON(fiber.Map{
			"is_valid" : false,
			"message" : "The orderTotal doesnt pass minimum order Value.",
		})
	}

	medicineIDs := make([]uuid.UUID, len(coupon_details.CartItems))
	fmt.Println("cart items are:%v",coupon_details.CartItems)
	for i, item := range coupon_details.CartItems {
			fmt.Println(item.ID, i)
			medicineIDs[i] = item.ID
	}
	fmt.Println("medicine id: %v", medicineIDs)

	fmt.Println("coupon code is:",coupon_details.CouponCode)
	medicineQuery := `SELECT medicine_id FROM coupon_medicine_map WHERE coupon_code=$1`
	rows, err = connPool.Query(ctx, medicineQuery, coupon_details.CouponCode)
	if err != nil {
		fmt.Println("Error retrieving medicine IDs")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error" : err.Error(),
		})
	}

	couponMedicineIDs := make(map[uuid.UUID]bool)
	for rows.Next(){
		var id uuid.UUID
		if err = rows.Scan(&id); err != nil {
			fmt.Println("Error scanning Medicine IDs.")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error" : err.Error(),
			})
		}
		couponMedicineIDs[id] = true;
	};

	var eligibleIds []uuid.UUID
	for _, id := range medicineIDs{
		if couponMedicineIDs[id]{
			eligibleIds = append(eligibleIds, id)
		}
	}
	fmt.Println(eligibleIds)

	medicinePriceQuery := `SELECT id,price FROM medicine WHERE id = ANY($1)`
	rows, err = connPool.Query(ctx, medicinePriceQuery, eligibleIds)
	var totalPrice, totalCharges float64
	for rows.Next(){
		var medicinePrice float64;
		var medicine_id uuid.UUID;
		if err = rows.Scan(&medicine_id,&medicinePrice); err != nil {
			fmt.Println("Error scanning coupon.")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error" : err.Error(),
			})
		}
		fmt.Println(medicine_id)
		if coupon_data.DiscountType == "flat" {
			if coupon_data.DiscountTarget == "inventory"{
				totalPrice = coupon_data.DiscountValue
				fmt.Println(totalPrice)
			} else if coupon_data.DiscountTarget == "charges" {
				totalCharges = coupon_data.DiscountValue
				fmt.Println(totalCharges)
			} else if coupon_data.DiscountTarget == "inventory_and_charges" {
				totalPrice = coupon_data.DiscountValue
				totalCharges = coupon_data.DiscountValue
				fmt.Println(totalPrice)
				fmt.Println(totalCharges)
			} else {
				continue
			}
		}else if coupon_data.DiscountType == "percentage" {
			if coupon_data.DiscountTarget == "inventory"{
				totalPrice += medicinePrice * (coupon_data.DiscountValue / 100)
				fmt.Println(totalPrice)
			} else if coupon_data.DiscountTarget == "charges" {
				totalCharges += medicinePrice * (coupon_data.DiscountValue / 100)
				fmt.Println(totalCharges)
			} else if coupon_data.DiscountTarget == "inventory_and_charges" {
				totalPrice += medicinePrice * (coupon_data.DiscountValue / 100)
				totalCharges += medicinePrice * (coupon_data.DiscountValue / 100)
				fmt.Println(totalPrice)
				fmt.Println(totalCharges)
			} else {
				continue
			}
		}
	}

	return c.JSON(fiber.Map{
		"is_valid" : true,
		"discount" : fiber.Map{
			"items_discount" : totalPrice,
			"charges_discount" : totalCharges,
		},
		"order_value_after_discount" : coupon_details.OrderTotal - (totalPrice + totalCharges),
		"message" : "Coupon applied succesfully",
	})
}

// @title Farmako Coupon API
// @version 1.0
// @description API for managing medicine coupons and discounts
// @contact.name API Support
// @contact.email support@pharmaapp.com
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:3000
// @BasePath /
func main(){
	db_url := "postgres://postgres:password@localhost:5432/coupondb"
	config, err := pgxpool.ParseConfig(db_url)

	if err != nil {
		log.Fatalf("Unable to parse config : %v", err)
	}

	connPool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Printf("unable to connect database: %v", err)
	}
	defer connPool.Close()
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e4,     
		MaxCost:     1 << 20, 
		BufferItems: 64,      
	})
	if err != nil {
		log.Fatalf("failed to create cache: %v", err)
	}

	app := fiber.New()
	app.Post("/admin/addCoupons", func(c *fiber.Ctx) error {
    return addCouponHandler(c, connPool)
	})

	app.Post("/coupon/update", func(c *fiber.Ctx) error {
    return updateCouponHandler(c,connPool)
	})
	
	app.Post("/coupon/applicable", func(c *fiber.Ctx) error {
    return getApplicableCoupons(c,connPool,cache)
	})

	app.Post("/coupon/validate",func(c *fiber.Ctx) error {
    return validateCouponHandler(c,connPool)
	})

	app.Get("/swagger/*", swagger.HandlerDefault)

	app.Listen(":3000")
}