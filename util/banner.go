package util

import "fmt"

const banner = `⚡ PIDARAS AUTOBUY ⚡
Спасибо за покупку! Вы алкаш ебаный!`

func PrintBanner() {
	fmt.Println()
	fmt.Println(banner)
	fmt.Println()
}
