package main

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"context"
	"log"
	"time"
	"github.com/google/uuid"
	"github.com/go-playground/validator/v10"
)

type Medicine struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Category string    `json:"category"`
	Price    float64   `json:"price"`
}

type CartItem struct {
	ID       string `json:"id"`
	Category string `json:"category"`
}

type OrderInput struct {
	CartItems  []Medicine `json:"cart_items"`
	OrderTotal float64   `json:"order_total"`
	Timestamp  time.Time `json:"timestamp"`
}

type ApplicableCoupon struct {
	CouponCode    string  `json:"coupon_code"`
	DiscountValue float64 `json:"discount_value"`
}

type ValidateCoupon struct {
	CouponCode string `json:"coupon_code"`
	OrderInput
}

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
	DiscountValue float64 `json:"discount_value" validate:"gt=0,lt=100"`
	DiscountTarget string `json:"discount_Target" validate:"required,oneof=inventory charges inventory_and_charges"`
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
		)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, couponData.CouponCode, couponData.ExpiryDate, couponData.UsageType, couponData.MinOrderValue, couponData.ValidFrom, couponData.ValidUntil, couponData.DiscountType, couponData.DiscountValue, couponData.MaxUsagePerUser, couponData.TermsAndConditions, couponData.DiscountTarget)
		if err != nil {
			fmt.Println("Error: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error" : err,
			})
		}

		medicineQuery := "INSERT INTO coupon_medicine_map(coupon_code, medicine_id) VALUES($1,$2)"
		categoryQuery := "INSERT INTO coupon_category_map(coupon_code, category_name) VALUES($1,$2)"

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
	})

	app.Post("/coupon/applicable", func(c *fiber.Ctx) error {
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

		medicine_query := "SELECT id,name,category,price from medicine WHERE id = ANY($1::uuid[])"
		ctx := context.Background()
		fmt.Println(medicineIDs)
		rows,err := connPool.Query(ctx, medicine_query, medicineIDs)

		if err != nil {
			fmt.Println("Error retreiving medicine Data")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error" : err,
			})
		}
		defer rows.Close()

		var medicines []uuid.UUID;
		var categories []string;

		for rows.Next(){
			var m Medicine;
			if err := rows.Scan(&m.ID, &m.Name, &m.Category, &m.Price); err != nil {
				fmt.Println("Error retreiving medicine")
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error" : err,
				})
			}
			medicines = append(medicines, m.ID)
			categories = append(categories, m.Category)
		}

		couponQuery := `SELECT DISTINCT c.coupon_code, c.discount_type, c.discount_value, c.min_order_value
		FROM coupon c
		LEFT JOIN coupon_medicine_map cmm ON c.coupon_code = cmm.coupon_code
		LEFT JOIN coupon_category_map ccm ON c.coupon_code = ccm.coupon_code
		WHERE cmm.medicine_id = ANY($1::uuid[]) OR ccm.category_name = ANY($2::text[])
		`

		var applicableCoupons []ApplicableCoupon
		rows,err = connPool.Query(ctx, couponQuery, medicines, categories)
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
	})

	app.Post("/coupon/validate", func (c *fiber.Ctx) error {
		var coupon_details ValidateCoupon
		if err := c.BodyParser(&coupon_details); err != nil {
			fmt.Println("Invalid Input")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error" : err,
			})
		}

		ctx := context.Background()
		couponQuery := `SELECT * FROM coupon WHERE coupon_code = $1`
		rows, err := connPool.Query(ctx, couponQuery, coupon_details.CouponCode)
		if err != nil {
			fmt.Println("Error retrieving Coupon details.")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error" : err.Error(),
			})
		}

		var coupon_data CouponData
		for rows.Next(){
			if err = rows.Scan(&coupon_data.CouponCode,&coupon_data.ExpiryDate,&coupon_data.UsageType,&coupon_data.MinOrderValue,&coupon_data.ValidFrom,&coupon_data.ValidUntil,&coupon_data.DiscountType,&coupon_data.DiscountValue,&coupon_data.MaxUsagePerUser,&coupon_data.TermsAndConditions,&coupon_data.DiscountTarget); err != nil {
				fmt.Println("Error scanning coupon.")
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error" : err.Error(),
				})
			}
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
    for i, item := range coupon_details.CartItems {
        medicineIDs[i] = item.ID
    }

		medicineQuery := `SELECT medicine_id FROM coupon_medicine_map WHERE coupon_code = $1`
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
			if coupon_data.DiscountType == "flat" {
				if coupon_data.DiscountTarget == "inventory"{
					totalPrice = coupon_data.DiscountValue
				} else if coupon_data.DiscountTarget == "charges" {
					totalCharges = coupon_data.DiscountValue
				} else if coupon_data.DiscountTarget == "inventory_and_charges" {
					totalPrice = coupon_data.DiscountValue
					totalCharges = coupon_data.DiscountValue
				} else {
					continue
				}
			}else if coupon_data.DiscountType == "percentage" {
				if coupon_data.DiscountTarget == "inventory"{
					totalPrice += medicinePrice * (coupon_data.DiscountValue / 100)
				} else if coupon_data.DiscountTarget == "charges" {
					totalCharges += medicinePrice * (coupon_data.DiscountValue / 100)
				} else if coupon_data.DiscountTarget == "inventory_and_charges" {
					totalPrice += medicinePrice * (coupon_data.DiscountValue / 100)
					totalCharges += medicinePrice * (coupon_data.DiscountValue / 100)
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
			"message" : "Coupon applied succesfully",
		})



	})
	app.Listen(":3000")
}