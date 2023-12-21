package user

import (
	"errors"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/gofiber/fiber/v2"
	"github.com/solanashuffle/backend/database"
	"go.mongodb.org/mongo-driver/bson"
	"math"
	"strconv"
)

// XPRequiredForLevel returns the amount of XP required to reach a certain level.
func XPRequiredForLevel(level uint64) float64 {
	return math.Pow(float64(level)/0.09, 1.5)
}

// XPNeededForLevel returns the amount of XP needed to reach the next level.
func XPNeededForLevel(level, currentXP uint64) float64 {
	return XPRequiredForLevel(level) - float64(currentXP)
}

// TotalXP returns the total amount of XP a user has.
func TotalXP(level, currentXP uint64) float64 {
	return float64(currentXP) + XPRequiredForLevel(level)
}

// LevelUp returns the level and XP of a user after the earned XP is added to the current XP.
func LevelUp(level, currentXP, earnedXP uint64) (uint64, uint64) {

	totalXP := TotalXP(level, currentXP) + float64(earnedXP)

	for totalXP >= XPRequiredForLevel(level+1) {
		level++
	}

	return level, uint64(totalXP - XPRequiredForLevel(level))
}

func HandleLevel(publicKey solana.PublicKey, earnedXP uint64) error {
	level, err := GetLevel(publicKey)
	if err != nil {
		return err
	}

	newLevel, experience, err := AddExperience(publicKey, earnedXP)
	if err != nil {
		return err
	}
	if newLevel > level {
		err = UpdateLevel(publicKey, newLevel, experience)
		if err != nil {
			return err
		}
	}
	return nil
}

func AddExperience(publicKey solana.PublicKey, earnedXP uint64) (uint64, uint64, error) {
	experience, err := GetExperience(publicKey)
	if err != nil {
		return 0, 0, err
	}

	level, err := GetLevel(publicKey)
	if err != nil {
		return 0, 0, err
	}

	level, experience = LevelUp(level, experience, earnedXP)

	return level, experience, nil
}

func UpdateLevel(publicKey solana.PublicKey, level uint64, experience uint64) error {
	needed := XPNeededForLevel(level + 1, experience)
	if err := database.UpdateOne(
		"users",
		bson.M{"publicKey": publicKey},
		bson.M{"$set": bson.M{"level": level, "experience": experience, "needed": needed}},
		false,
	); err != nil {
		return err
	}
	return nil
}

//HandleLevelSET Set Level
func HandleLevelSET(c *fiber.Ctx) error {
	publicKey, err := solana.PublicKeyFromBase58(c.Params("+"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	body := new(ProfileBody)
	err = c.BodyParser(body)
	if err != nil {
		fmt.Println(err)
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	if body.Stats.Level.XP < 0 {
		return JSONError(c, fiber.StatusBadRequest, errors.New("invalid XP"))
	}

	ui64, err := strconv.ParseUint(strconv.FormatUint(body.Stats.Level.XP, 10), 10, 64)

	err = HandleLevel(publicKey, ui64)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	level, err := GetLevel(publicKey)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	return c.Status(fiber.StatusOK).JSON(level)
}

func GetLevel(publicKey solana.PublicKey) (uint64, error) {
	user, err := database.GetUser(publicKey)
	if err != nil {
		return 0, err
	}
	return user.Stats.Level.Value, nil
}

func GetExperience(publicKey solana.PublicKey) (uint64, error) {
	user, err := database.GetUser(publicKey)
	if err != nil {
		return 0, err
	}
	return user.Stats.Level.XP, nil
}