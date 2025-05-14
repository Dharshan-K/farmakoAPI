package main

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"context"
	"log"
	"time"
	"github.com/go-playground/validator/v10"
)

// the Coupon struct handles and validates the coupon data
type CouponData struct {
	CouponCode string `json:"coupon_code" validate:"required,min=3,max=50"`
	ExpiryDate time.Time `json:"expiry_date" validate:"required"`
	ApplicableMedicineId []string `json:"applicable_medicine_id" validate="dive,uuid4`
	ApplicableCategories []string `json:"applicable_categories" validate="dive,required`
	UsageType string `json:"usage_type" validate:"required,oneof=one_time multi_use time_based"`
	MinOrderValue float64 `json:"min_order_value" validate:"gte=0"`
	ValidFrom time.Time `json:"valid_from" validate:"required"`
	ValidUntil time.Time `json:"valid_until" validate:"required,gtfield=ValidFrom"`
	TermsAndConditions string `json:"terms_and_conditions"`
	DiscountType string `json:"discount_type" validate:"required,oneof=flat percentage"`
	DiscountValue float64 `json:"discount_value" validate:"gt=0"`
	MaxUsagePerUser int `json:"max_usage_per_user" validate:"gt=0"`
}


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

	app := fiber.New()
	var validate = validator.New()

	app.Post("/admin/addCoupons", func(c *fiber.Ctx) error {
		var couponData CouponData;

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

		ctx := context.Background()
		_, err := connPool.Exec(ctx, `INSERT INTO coupon(coupon_code,
			expiry_date,
			usage_type,
			min_order_value,
			valid_from,
			valid_until,
			discount_type,
			discount_value,
			max_usage_per_user,
			terms_and_conditions
	)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, couponData.CouponCode, couponData.ExpiryDate, couponData.UsageType, couponData.MinOrderValue, couponData.ValidFrom, couponData.ValidUntil, couponData.DiscountType, couponData.DiscountValue, couponData.MaxUsagePerUser, couponData.TermsAndConditions)
	if err != nil {
		fmt.Println("Error: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error" : err,
		})
	}

	return c.SendString("Coupon added successfully")		
	})

	app.Listen(":3000")
}