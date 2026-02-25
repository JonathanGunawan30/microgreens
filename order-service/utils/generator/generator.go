package generator

import (
	"fmt"
	"math/rand"
	"time"
)

func GenerateOrderCode() string {
	return fmt.Sprintf("ORD-%s-%d", time.Now().Format("20060102150405"), rand.Intn(1000000))
}
