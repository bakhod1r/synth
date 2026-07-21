package providers

import (
	"fmt"

	"github.com/bakhodir/synth/schema"
)

// Third catalog batch: e-commerce, DevOps, finance and media types.

func init() {
	set(schema.KindOrderStatus, []string{"pending", "paid", "processing", "shipped", "delivered", "cancelled", "refunded", "returned"})
	set(schema.KindNickname, []string{"Ace", "Buddy", "Champ", "Chief", "Duke", "Ghost", "Maverick", "Ninja", "Rocket", "Shadow", "Spike", "Turbo", "Viper", "Wizard"})
	set(schema.KindCocktail, []string{"Margarita", "Mojito", "Martini", "Negroni", "Old Fashioned", "Cosmopolitan", "Daiquiri", "Manhattan", "Mai Tai", "Piña Colada"})
	set(schema.KindCoffee, []string{"Espresso", "Latte", "Cappuccino", "Americano", "Macchiato", "Mocha", "Flat White", "Cortado", "Ristretto", "Affogato"})
	set(schema.KindSuperhero, []string{"Spider-Man", "Iron Man", "Batman", "Superman", "Wonder Woman", "Thor", "Hulk", "Captain America", "Black Panther", "Doctor Strange"})
	set(schema.KindPetName, []string{"Max", "Bella", "Charlie", "Luna", "Rocky", "Lucy", "Milo", "Daisy", "Simba", "Coco", "Buddy", "Ruby"})
	set(schema.KindLogLevel, []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL"})
	set(schema.KindEnvironment, []string{"development", "staging", "production", "test", "qa", "sandbox"})
	set(schema.KindAWSRegion, []string{"us-east-1", "us-west-2", "eu-west-1", "eu-central-1", "ap-south-1", "ap-southeast-1", "ap-northeast-1", "sa-east-1"})
	set(schema.KindCloudProvider, []string{"AWS", "Google Cloud", "Azure", "DigitalOcean", "Cloudflare", "Oracle Cloud", "Linode", "Vultr"})
	set(schema.KindContainerImage, []string{"nginx:latest", "postgres:16", "redis:7", "node:20-alpine", "python:3.12", "golang:1.22", "ubuntu:22.04", "alpine:3.19"})
	set(schema.KindHTTPHeader, []string{"Content-Type", "Authorization", "Accept", "User-Agent", "Cache-Control", "Cookie", "Host", "Referer", "X-Request-Id", "ETag"})
	set(schema.KindKeyboardKey, []string{"Enter", "Escape", "Tab", "Space", "Backspace", "Shift", "Ctrl", "Alt", "Delete", "ArrowUp", "F1", "CapsLock"})
	set(schema.KindMusicNote, []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"})
	set(schema.KindMedal, []string{"Gold", "Silver", "Bronze"})
	set(schema.KindTShirtSize, []string{"XS", "S", "M", "L", "XL", "XXL", "XXXL"})
	set(schema.KindPriority, []string{"Low", "Medium", "High", "Critical"})

	registry[schema.KindIMEI] = func(c Ctx) any {
		body := c.Rand.Digits(14)
		return body + string(luhnCheck(body))
	}
	registry[schema.KindUPC] = func(c Ctx) any {
		body := c.Rand.Digits(11)
		return body + string(upcCheck(body))
	}
	registry[schema.KindRoutingNumber] = func(c Ctx) any { return c.Rand.Digits(9) }
	registry[schema.KindAccountNumber] = func(c Ctx) any { return c.Rand.Digits(c.Rand.IntRange(8, 12)) }
	registry[schema.KindErrorCode] = func(c Ctx) any {
		return fmt.Sprintf("ERR_%s_%s", upperLetters(c.Rand, 3), c.Rand.Digits(4))
	}
	registry[schema.KindCron] = func(c Ctx) any {
		return fmt.Sprintf("%d %d * * %d", c.Rand.Intn(60), c.Rand.Intn(24), c.Rand.Intn(7))
	}
	registry[schema.KindFileSize] = func(c Ctx) any {
		units := []string{"KB", "MB", "GB"}
		return fmt.Sprintf("%d.%d %s", c.Rand.IntRange(1, 999), c.Rand.Intn(10), units[c.Rand.Pick(len(units))])
	}
	registry[schema.KindDuration] = func(c Ctx) any {
		return fmt.Sprintf("%dh %dm", c.Rand.Intn(24), c.Rand.Intn(60))
	}
	registry[schema.KindGitTag] = func(c Ctx) any {
		return fmt.Sprintf("v%d.%d.%d", c.Rand.Intn(5), c.Rand.Intn(20), c.Rand.Intn(50))
	}
	registry[schema.KindCouponCode] = func(c Ctx) any {
		promos := []string{"SAVE", "DEAL", "OFF", "SALE", "PROMO", "GET"}
		return pick(c.Rand, promos) + c.Rand.Digits(2) + "-" + upperAlnum(c.Rand, 4)
	}
}

// upcCheck returns the UPC-A check digit for an 11-digit body.
func upcCheck(body string) byte {
	sum := 0
	for i := 0; i < len(body); i++ {
		d := int(body[i] - '0')
		if i%2 == 0 {
			d *= 3
		}
		sum += d
	}
	return byte('0' + (10-sum%10)%10)
}
