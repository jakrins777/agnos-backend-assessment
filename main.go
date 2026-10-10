package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ==========================================
// 1. Models (Database Schema ด้วย GORM)
// ==========================================

// Task 3: Model สำหรับ Staff
type Staff struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;not null;size:100" json:"username"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Hospital     string    `gorm:"index;not null;size:100" json:"hospital"`
	CreatedAt    time.Time `json:"created_at"`
}

// Task 2: Model สำหรับ Patient ที่รองรับโครงสร้างข้อมูลโรงพยาบาล
type Patient struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Hospital     string    `gorm:"index;not null;size:100" json:"hospital"`
	PatientHN    string    `gorm:"size:50" json:"patient_hn"`
	FirstNameTH  string    `gorm:"size:100" json:"first_name_th"`
	MiddleNameTH string    `gorm:"size:100" json:"middle_name_th"`
	LastNameTH   string    `gorm:"size:100" json:"last_name_th"`
	FirstNameEN  string    `gorm:"size:100" json:"first_name_en"`
	MiddleNameEN string    `gorm:"size:100" json:"middle_name_en"`
	LastNameEN   string    `gorm:"size:100" json:"last_name_en"`
	DateOfBirth  string    `gorm:"type:date" json:"date_of_birth"`
	NationalID   string    `gorm:"index;size:50" json:"national_id"`
	PassportID   string    `gorm:"index;size:50" json:"passport_id"`
	PhoneNumber  string    `gorm:"size:50" json:"phone_number"`
	Email        string    `gorm:"size:100" json:"email"`
	Gender       string    `gorm:"size:1" json:"gender"` // "M", "F"
	CreatedAt    time.Time `json:"created_at"`
}

// ==========================================
// 2. DTOs & Claims
// ==========================================

type StaffCreateRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Hospital string `json:"hospital" binding:"required"`
}

type StaffLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Hospital string `json:"hospital" binding:"required"`
}

type PatientSearchQuery struct {
	NationalID  string `form:"national_id"`
	PassportID  string `form:"passport_id"`
	FirstName   string `form:"first_name"`
	MiddleName  string `form:"middle_name"`
	LastName    string `form:"last_name"`
	DateOfBirth string `form:"date_of_birth"`
	PhoneNumber string `form:"phone_number"`
	Email       string `form:"email"`
}

type JWTClaims struct {
	Username string `json:"username"`
	Hospital string `json:"hospital"`
	jwt.RegisteredClaims
}

var (
	db        *gorm.DB
	jwtSecret = []byte("AgnosSecretKey2026")
)

// ==========================================
// 3. Database Connection
// ==========================================

func initDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// ค่า default สำหรับรันในเครื่อง (ปรับ user/password/dbname ตามเครื่องของคุณ)
		dsn = "host=localhost user=postgres password=postgres dbname=agnos_db port=5432 sslmode=disable TimeZone=Asia/Bangkok"
	}

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("เชื่อมต่อฐานข้อมูลล้มเหลว: %v", err)
	}

	// สร้างตารางใน PostgreSQL อัตโนมัติจาก Struct
	err = db.AutoMigrate(&Staff{}, &Patient{})
	if err != nil {
		log.Fatalf("AutoMigrate ล้มเหลว: %v", err)
	}
	log.Println("เชื่อมต่อฐานข้อมูลและ Migrate ตารางสำเร็จ")
}

// ==========================================
// 4. Main Server & Routing
// ==========================================

func main() {
	initDB()

	r := gin.Default()
	r.SetTrustedProxies(nil)

	// API บุคลากรโรงพยาบาล
	r.POST("/staff/create", createStaffHandler)
	r.POST("/staff/login", staffLoginHandler)

	// Protected API: ต้อง Login และแนบ Token ถึงจะค้นหาได้
	authGroup := r.Group("/")
	authGroup.Use(authMiddleware())
	{
		authGroup.GET("/patient/search", searchPatientHandler)
	}

	// จำลอง Mock HIS ของ Hospital A สำหรับข้อ 1
	r.GET("/mock/hospital-a/patient/search/:id", mockHospitalAHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	r.Run(":" + port)
}

// ==========================================
// 5. Handlers
// ==========================================

// Task 4: สร้างเจ้าหน้าที่ใหม่ (/staff/create)
func createStaffHandler(c *gin.Context) {
	var req StaffCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ข้อมูลไม่ถูกต้องหรือกรอกไม่ครบ"})
		return
	}

	// แฮชรหัสผ่านด้วย bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "เข้ารหัสผ่านไม่สำเร็จ"})
		return
	}

	staff := Staff{
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Hospital:     req.Hospital,
	}

	if err := db.Create(&staff).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username นี้มีผู้ใช้งานแล้ว"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "สร้างบัญชีเจ้าหน้าที่สำเร็จ"})
}

// Task 4: เจ้าหน้าที่เข้าสู่ระบบ (/staff/login)
func staffLoginHandler(c *gin.Context) {
	var req StaffLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ข้อมูลไม่ถูกต้อง"})
		return
	}

	var staff Staff
	// ค้นหา username และ hospital ที่ตรงกัน
	if err := db.Where("username = ? AND hospital = ?", req.Username, req.Hospital).First(&staff).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ชื่อผู้ใช้หรือโรงพยาบาลไม่ถูกต้อง"})
		return
	}

	// ตรวจสอบความถูกต้องของรหัสผ่าน
	if err := bcrypt.CompareHashAndPassword([]byte(staff.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "รหัสผ่านไม่ถูกต้อง"})
		return
	}

	// สร้าง JWT Token
	claims := JWTClaims{
		Username: staff.Username,
		Hospital: staff.Hospital,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ออก Token ไม่สำเร็จ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// Task 4: ค้นหาผู้ป่วย (/patient/search) - ค้นหาเฉพาะโรงพยาบาลของ Staff
func searchPatientHandler(c *gin.Context) {
	staffHospital, _ := c.Get("staff_hospital")

	var q PatientSearchQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// เริ่มสร้าง Query และล็อกเงื่อนไขให้ค้นหาได้เฉพาะโรงพยาบาลของ Staff เท่านั้น
	query := db.Model(&Patient{}).Where("hospital = ?", staffHospital)

	// เพิ่มเงื่อนไขการค้นหาแบบ Dynamic ถ้ามีการส่ง Query Parameter มา
	if q.NationalID != "" {
		query = query.Where("national_id = ?", q.NationalID)
	}
	if q.PassportID != "" {
		query = query.Where("passport_id = ?", q.PassportID)
	}
	if q.FirstName != "" {
		query = query.Where("first_name_th ILIKE ? OR first_name_en ILIKE ?", "%"+q.FirstName+"%", "%"+q.FirstName+"%")
	}
	if q.MiddleName != "" {
		query = query.Where("middle_name_th ILIKE ? OR middle_name_en ILIKE ?", "%"+q.MiddleName+"%", "%"+q.MiddleName+"%")
	}
	if q.LastName != "" {
		query = query.Where("last_name_th ILIKE ? OR last_name_en ILIKE ?", "%"+q.LastName+"%", "%"+q.LastName+"%")
	}
	if q.DateOfBirth != "" {
		query = query.Where("date_of_birth = ?", q.DateOfBirth)
	}
	if q.PhoneNumber != "" {
		query = query.Where("phone_number = ?", q.PhoneNumber)
	}
	if q.Email != "" {
		query = query.Where("email = ?", q.Email)
	}

	var patients []Patient
	if err := query.Find(&patients).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "เกิดข้อผิดพลาดในการค้นหา"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"hospital": staffHospital,
		"count":    len(patients),
		"data":     patients,
	})
}

// Middleware สำหรับแกะ JWT และดึงค่า Hospital ของ Staff
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "กรุณาแนบ Authorization Bearer Token"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &JWTClaims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token ไม่ถูกต้องหรือหมดอายุ"})
			return
		}

		c.Set("staff_username", claims.Username)
		c.Set("staff_hospital", claims.Hospital)
		c.Next()
	}
}

// Mock HIS Server สำหรับ Hospital A
func mockHospitalAHandler(c *gin.Context) {
	id := c.Param("id")
	if id == "1234567890123" {
		c.JSON(http.StatusOK, gin.H{
			"first_name_th": "สมชาย",
			"last_name_th":  "ใจดี",
			"patient_hn":    "HN001",
			"national_id":   "1234567890123",
			"gender":        "M",
		})
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบผู้ป่วย"})
}
