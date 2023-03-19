package controller

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Kruzikael014/oldegg-backend/config"
	"github.com/Kruzikael014/oldegg-backend/model"
	"github.com/Kruzikael014/oldegg-backend/util"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func GetUsers(c *gin.Context) {
	users := []model.User{}
	config.DB.Find(&users)
	c.JSON(200, &users)
}

func GetUser(c *gin.Context) {
	var user model.User
	config.DB.First(&user, "email = ?", c.PostForm("email"))
	if user.ID == 0 {
		c.JSON(http.StatusOK, gin.H{
			"error": "User not found",
		})
		return
	}
	c.JSON(200, &user)
}

func GetUserById(c *gin.Context) {
	var userId = c.Param("id")
	var user model.User
	config.DB.First(&user, "id = ?", userId)
	c.JSON(200, user)
}

func CreateUser(c *gin.Context) {
	var user model.User
	c.BindJSON(&user)

	var countEmail int64 = 0

	config.DB.Model(model.User{}).Where("email = ?", user.Email).Count(&countEmail)

	// not unique, found 1 data in database that equals our email (already exists)
	if countEmail != 0 {
		c.JSON(http.StatusOK, gin.H{
			"error": "Email is not unique",
		})
		return
	}

	// invalid email
	if !util.IsValidEmail(user.Email) {
		c.JSON(http.StatusOK, gin.H{
			"error": "Email is not valid",
		})
		return
	}

	if len(user.Phone_num) != 0 {
		// invalid phone number
		if !util.IsValidPhone(user.Phone_num) {
			c.JSON(http.StatusOK, gin.H{
				"error": "Phone number is not valid",
			})
			return
		}

		var countPhoneNum int64 = 0
		// not unique phone number
		config.DB.Model(model.User{}).Where("phone_num = ?", user.Phone_num).Count(&countPhoneNum)

		if countPhoneNum != 0 {
			c.JSON(http.StatusOK, gin.H{
				"error": "Phone number is not unique",
			})
			return
		}
	}

	// password contain 1 lowercase, 1 uppercase, numbers, 1 special char, and has length 8 - 30
	if !util.IsValidPassword(user.Password) {
		c.JSON(http.StatusOK, gin.H{
			"error": "Password is not valid",
		})
		return
	}

	// Hash user password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": "Failed to hash password",
		})
		return
	}
	user.Password = fmt.Sprintf("%s", hashedPassword)
	config.DB.Create(&user)
	if user.Role_name == "Shop Owner" {
		config.DB.Create(&model.Shop{ID: user.ID, Average_Rating: 0, Number_of_Sales: 0})
		SendMessage("Your shop successfully created!", []string{user.Email})
	}
	c.JSON(200, &user)
}

func DeleteUser(c *gin.Context) {
	var user model.User
	config.DB.Where("id = ?", c.Param("id")).Delete(&user)
	c.JSON(200, &user)
}

func UpdateUser(c *gin.Context) {
	var user model.User
	config.DB.Where("id = ?", c.PostForm("update_id")).First(&user)
	c.BindJSON(&user)
	hashedPassword, err := util.Encrypt(c.PostForm("password"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": "Failed to hash password",
		})
		return
	}
	user.Password = hashedPassword
	config.DB.Save(&user)
	if !c.Writer.Written() {
		c.JSON(200, &user)
	}
}

func Login(c *gin.Context) {
	var user model.User
	var body struct {
		Email    string
		Password string
	}
	if c.Bind(&body) != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": "Failed to read body",
		})
		return
	}
	config.DB.First(&user, "email = ?", body.Email)
	if user.ID == 0 {
		c.JSON(http.StatusOK, gin.H{
			"error": "Account not found",
		})
		return
	}
	err := util.Decrypt(body.Password, user.Password)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": "Invalid password",
		})
		return
	}
	if user.Status != "Clear" {
		c.JSON(http.StatusOK, gin.H{
			"error": "Account has been banned",
		})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.Email,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": "Failed to generate token",
		})
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("JWToken", tokenString, 3600*24, "", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
	})
}

func Validate(c *gin.Context) {
	user, _ := c.Get("user")
	c.JSON(http.StatusOK, gin.H{
		"message": user,
	})
}

func GetCustomer(c *gin.Context) {
	user := []model.User{}
	config.DB.Where("role_name = 'Customer'").Find(&user)
	c.JSON(200, user)
}

func ComparePass(c *gin.Context) {
	var request struct {
		UserID   uint   `json:"user_id"`
		Password string `json:"password"`
	}
	c.ShouldBindJSON(&request)
	var user model.User
	config.DB.First(&user, "id = ?", request.UserID)
	err := util.Decrypt(request.Password, user.Password)
	if err != nil {
		c.String(200, "Failed")
		return
	}
	c.String(200, "Success")
}

func GetShop(c *gin.Context) {
	user := []model.User{}
	config.DB.Where("role_name = 'Shop Owner'").Find(&user)
	c.JSON(200, user)
}

// BanUser sets the user's status to "Banned"
func BanUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user model.User
	if err := config.DB.Where("id = ?", id).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.Status = "Banned"
	if err := config.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	SendMessage("You account has been banned, check the rules and procedure for more information", []string{user.Email})
	c.JSON(http.StatusOK, user)
}

func UnbanUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user model.User
	if err := config.DB.Where("id = ?", id).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.Status = "Clear"
	if err := config.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	SendMessage("You account has been unbanned, now you can do your things as usual", []string{user.Email})
	c.JSON(http.StatusOK, user)
}

func ChangePassword(c *gin.Context) {
	var changePassRequest struct {
		Email       string `json:"email"`
		NewPassword string `json:"new_password"`
	}
	c.ShouldBindJSON(&changePassRequest)

	var user model.User

	config.DB.First(&user, "email = ?", changePassRequest.Email)

	hashedPassword, err := util.Encrypt(changePassRequest.NewPassword)

	if err != nil {
		c.String(200, "Failed to hash the password")
		return
	}

	user.Password = hashedPassword

	config.DB.Save(&user)
	c.String(200, "OK")

}

func ChangePhoneNum(c *gin.Context) {
	var Request struct {
		UserID      uint   `json:"user_id"`
		NewPhoneNum string `json:"phone_number"`
	}
	c.ShouldBindJSON(&Request)
	if !util.IsValidPhone(Request.NewPhoneNum) {
		c.String(200, "Phone number is invalid")
		return
	}
	var user model.User
	err := config.DB.First(&user, "id = ?", Request.UserID).Error
	if err != nil {
		c.String(200, "Failed to find the user!")
		return
	}
	user.Phone_num = Request.NewPhoneNum
	err = config.DB.Save(&user).Error
	if err != nil {
		c.String(200, "Failed to aplpy the changes!")
		return
	}
	c.String(200, "Phone number successfully updated!")
}

func DecreaseBalance(c *gin.Context) {
	var DecreaseBalanceRequest struct {
		UserId      uint `json:"user_id"`
		TotalAmount uint `json:"total_amount"`
	}
	c.ShouldBindJSON(&DecreaseBalanceRequest)
	var user model.User
	config.DB.First(&user, "id = ?", DecreaseBalanceRequest.UserId)
	if user.Balance < DecreaseBalanceRequest.TotalAmount {
		c.String(200, "You dont have enough balance!")
		return
	}
	user.Balance -= DecreaseBalanceRequest.TotalAmount
	config.DB.Save(&user)
	c.String(200, "Balance successfully charged!")
}
