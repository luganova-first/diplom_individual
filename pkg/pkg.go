package pkg

import (
	"strconv"
	"strings"
)

// Проверка номера заказа с помощью алгоритма Луна
func ValidateLuhn(number string) bool {
	// number := strconv.FormatInt(number, 10)

	// Удаляем пробелы и дефисы
	number = strings.ReplaceAll(number, " ", "")
	number = strings.ReplaceAll(number, "-", "")

	// Проверяем, что строка не пустая и состоит только из цифр
	if len(number) == 0 {
		return false
	}

	// Преобразуем строку в срез цифр
	digits := make([]int, len(number))
	for i, ch := range number {
		if ch < '0' || ch > '9' {
			return false
		}
		digits[i], _ = strconv.Atoi(string(ch))
	}

	// Алгоритм Луна
	var sum int
	isSecond := false

	// Идём справа налево
	for i := len(digits) - 1; i >= 0; i-- {
		digit := digits[i]

		if isSecond {
			digit *= 2
			if digit > 9 {
				digit = digit%10 + 1
			}
		}

		sum += digit
		isSecond = !isSecond
	}

	return sum%10 == 0
}
