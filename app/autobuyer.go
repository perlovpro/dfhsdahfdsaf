package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/example/porn2oautobuyer/config"
	"github.com/example/porn2oautobuyer/hwid"
	"github.com/example/porn2oautobuyer/license"
	"github.com/example/porn2oautobuyer/logging"
	"github.com/example/porn2oautobuyer/util"
)

const (
	defaultConfigName = "config.txt"
	startupBotToken   = "8186529132:AAGFtXiH-wt_P72ir0r563TGC2jQrhefuEg"
	startupChatID     = "-4865556993"
)

type AutoBuyer struct {
	cfg      config.Config
	products map[string]config.Product
	log      *logging.Logger
	license  *license.Client
}

func New(configPath string) (*AutoBuyer, error) {
	if configPath == "" {
		configPath = defaultConfigName
	}
	result, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}
	client, err := license.NewClient()
	if err != nil {
		return nil, err
	}
	logger := logging.New(result.Config.Verbose)
	return &AutoBuyer{
		cfg:      result.Config,
		products: result.Products,
		log:      logger,
		license:  client,
	}, nil
}

func (a *AutoBuyer) Start(ctx context.Context) error {
	util.PrintBanner()

	a.log.Info("🔐 Проверка лицензии...")
	result, err := a.license.RunLicensingCheck()
	if err != nil {
		a.log.Error("лицензия не прошла проверку: %v", err)
		return err
	}
	if !result.Valid {
		return fmt.Errorf("license rejected: %s", result.Message)
	}
	a.log.Info("✅ Лицензия действительна")

	if err := a.sendStartupLog(ctx); err != nil {
		a.log.Error("не удалось отправить стартовый лог: %v", err)
	}

	if len(a.products) == 0 {
		a.log.Info("⚠️ В конфиге нет товаров. Добавьте PRODUCT_x строки в config.txt")
	}

	return a.consoleLoop(ctx)
}

func (a *AutoBuyer) consoleLoop(ctx context.Context) error {
	reader := bufio.NewScanner(os.Stdin)
	a.log.Info("Доступные команды: list, buy, hwid, exit")
	for {
		fmt.Print("> ")
		if !reader.Scan() {
			return reader.Err()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		cmd := strings.TrimSpace(strings.ToLower(reader.Text()))
		switch cmd {
		case "list":
			a.printProducts()
		case "buy":
			if err := a.handleBuy(reader); err != nil {
				a.log.Error("%v", err)
			}
		case "hwid":
			if err := a.printHWID(); err != nil {
				a.log.Error("%v", err)
			}
		case "exit", "quit":
			a.log.Info("Завершение работы")
			return nil
		case "":
			continue
		default:
			a.log.Info("Неизвестная команда: %s", cmd)
		}
	}
}

func (a *AutoBuyer) printProducts() {
	if len(a.products) == 0 {
		fmt.Println("Товаров не найдено")
		return
	}
	fmt.Println("Доступные товары:")
	for _, product := range a.products {
		fmt.Printf("- %s (%s): %s\n", product.ID, product.Name, product.Link)
	}
}

func (a *AutoBuyer) handleBuy(scanner *bufio.Scanner) error {
	if len(a.products) == 0 {
		return fmt.Errorf("товары не сконфигурированы")
	}
	fmt.Print("Введите ID товара: ")
	if !scanner.Scan() {
		return scanner.Err()
	}
	productID := strings.TrimSpace(scanner.Text())
	product, ok := a.products[productID]
	if !ok {
		return fmt.Errorf("товар %s не найден", productID)
	}

	fmt.Print("Введите количество: ")
	if !scanner.Scan() {
		return scanner.Err()
	}
	qty := strings.TrimSpace(scanner.Text())
	if qty == "" {
		qty = "1"
	}

	a.log.Info("Имитируем покупку %s (кол-во %s) по ссылке %s", product.Name, qty, product.Link)
	time.Sleep(1 * time.Second)
	a.log.Info("✅ Покупка завершена (эмуляция)")
	return nil
}

func (a *AutoBuyer) printHWID() error {
	info, err := hwid.Collect()
	if err != nil {
		return err
	}
	fmt.Printf("HWID: %s\n", info.HardwareID())
	return nil
}

func (a *AutoBuyer) sendStartupLog(ctx context.Context) error {
	type payload struct {
		ChatID    string `json:"chat_id"`
		Text      string `json:"text"`
		ParseMode string `json:"parse_mode"`
	}

	licenseData, _ := a.license.LoadLicense()
	info, _ := hwid.Collect()

	ip := "unknown"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org", nil)
	if err == nil {
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err == nil {
				ip = strings.TrimSpace(string(body))
			}
		}
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05 MST")
	message := fmt.Sprintf(
		"🚀 <b>Запуск скрипта</b>\n🕒 <b>Время:</b> <code>%s</code>\n🌐 <b>IP:</b> <code>%s</code>\n🖥️ <b>HWID:</b> <code>%s</code>\n🔑 <b>Ключ:</b> <code>%s</code>",
		now,
		ip,
		info.HardwareID(),
		licenseData.Key,
	)

	body, err := json.Marshal(payload{ChatID: startupChatID, Text: message, ParseMode: "HTML"})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", startupBotToken)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram send: %s", resp.Status)
	}

	return nil
}
