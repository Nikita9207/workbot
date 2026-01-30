package bot

import (
	"strings"
)

// isPowerliftingSport проверяет, является ли вид спорта силовым
func isPowerliftingSport(sport string) bool {
	sportLower := strings.ToLower(sport)
	powerliftingSports := []string{
		"пауэрлифтинг", "жим лёжа", "жим лежа", "становая тяга",
		"присед", "приседания", "троеборье",
	}
	for _, s := range powerliftingSports {
		if strings.Contains(sportLower, s) {
			return true
		}
	}
	return false
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
