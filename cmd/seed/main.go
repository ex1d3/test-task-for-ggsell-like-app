package main

import (
	"context"
	"errors"
	"fmt"
	"gg-sell-like-core/internal/key"
	"gg-sell-like-core/internal/platform"
	"gg-sell-like-core/internal/product"
	"gg-sell-like-core/pkg/currency"
	"gg-sell-like-core/pkg/transactor"
	"log/slog"
	"os"
	"time"
)

func main() {
	baseLog := platform.NewLogger()
	seedLog := baseLog.With(
		"component", "seeder",
	)
	seedLog.Info("started")
	if err := run(seedLog, baseLog); err != nil {
		seedLog.Error("run", "err", err)
		os.Exit(1)
	}
	seedLog.Info("completed")
}

func run(seedLog *slog.Logger, baseLog *slog.Logger) error {
	ctx := context.Background()

	cfg := platform.NewSeedConfig()
	db, err := platform.NewPostgreSQL(ctx, cfg.DatabaseURL, baseLog)
	if err != nil {
		return fmt.Errorf("init postgre: %w", err)
	}
	defer db.Close()

	tx := transactor.NewPostgreTransactor(db)

	now := time.Now().UTC()

	productRepo := product.NewPostgreRepository(db)
	keyRepo := key.NewPostgreRepository(db)

	products := []product.Product{
		product.NewProduct(
			"STEAM-TOPUP-500",
			"Пополнение Steam 500 ₽",
			product.TypeTopUp,
			500,
			currency.RUB,
			"assets/steam.png",
			now,
		),
		product.NewProduct(
			"STEAM-TOPUP-1000",
			"Пополнение Steam 1000 ₽",
			product.TypeTopUp,
			1000,
			currency.RUB,
			"assets/steam.png",
			now,
		),
		product.NewProduct(
			"STEAM-TOPUP-2500",
			"Пополнение Steam 2500 ₽",
			product.TypeTopUp,
			2500,
			currency.RUB,
			"assets/steam.png",
			now,
		),
		product.NewProduct(
			"KEY-CS2-PRIME",
			"CS2 Prime Status ключ",
			product.TypeKey,
			1290,
			currency.RUB,
			"assets/cs2.png",
			now,
		),
		product.NewProduct(
			"KEY-GTA5",
			"GTA V ключ активации",
			product.TypeKey,
			1990,
			currency.RUB,
			"assets/gta5.png",
			now,
		),
		product.NewProduct(
			"KEY-EFT",
			"Escape from Tarkov ключ",
			product.TypeKey,
			3490,
			currency.RUB,
			"assets/eft.png",
			now,
		),
		product.NewProduct(
			"SUB-DISCORD-1M",
			"Discord Nitro 1 месяц",
			product.TypeSubscription,
			399,
			currency.RUB,
			"assets/discord.png",
			now,
		),
		product.NewProduct(
			"SUB-YT-3M",
			"YouTube Premium 3 месяца",
			product.TypeSubscription,
			1490,
			currency.RUB,
			"assets/youtube.png",
			now,
		),
		product.NewProduct(
			"SUB-SPOTIFY-1M",
			"Spotify Premium 1 месяц",
			product.TypeSubscription,
			299,
			currency.RUB,
			"assets/spotify.png",
			now,
		),
		product.NewProduct(
			"GIFT-PSN-1000",
			"PlayStation Store карта 1000 ₽",
			product.TypeGiftCard,
			1000,
			currency.RUB,
			"assets/psn.png",
			now,
		),
		product.NewProduct(
			"GIFT-XBOX-1500",
			"Xbox Gift Card 1500 ₽",
			product.TypeGiftCard,
			1500,
			currency.RUB,
			"assets/xbox.png",
			now,
		),
		product.NewProduct(
			"GIFT-ROBLOX-800",
			"Roblox 800 Robux",
			product.TypeGiftCard,
			890,
			currency.RUB,
			"assets/roblox.png",
			now,
		),
	}
	keys := []key.Key{
		key.NewKey("KEY-CS2-PRIME", "LFXC-TNCS-BPCD", now),
		key.NewKey("KEY-GTA5", "P3EI-W8UO-9B4K", now),
		key.NewKey("KEY-EFT", "FEL3-GUXN-TCCH", now),
		key.NewKey("KEY-GTA5", "YPLV-QK2Z-IUS5", now),
		key.NewKey("KEY-CS2-PRIME", "0K9E-P1FR-BY1U", now),
		key.NewKey("KEY-EFT", "5LZV-UQ48-RXCZ", now),
		key.NewKey("KEY-CS2-PRIME", "X93K-NYAQ-GEC1", now),
		key.NewKey("KEY-GTA5", "EIO5-CQT5-35KO", now),
		key.NewKey("KEY-EFT", "M58F-GIIR-VJAP", now),
		key.NewKey("KEY-GTA5", "NU8Y-SWYB-6252", now),
		key.NewKey("KEY-CS2-PRIME", "OODW-CCHF-MBAF", now),
		key.NewKey("KEY-EFT", "DNA5-WFJM-NE49", now),
		key.NewKey("KEY-EFT", "QRDD-MJ3F-A8TF", now),
		key.NewKey("KEY-CS2-PRIME", "TAT9-5ZJN-G1T2", now),
		key.NewKey("KEY-GTA5", "LI39-4330-ISMB", now),
		key.NewKey("KEY-GTA5", "BKJY-8Q79-8NHI", now),
		key.NewKey("KEY-EFT", "HHW6-4RX2-DX62", now),
		key.NewKey("KEY-CS2-PRIME", "1RG2-L28O-O80G", now),
		key.NewKey("KEY-GTA5", "EF63-F39X-MTEA", now),
		key.NewKey("KEY-EFT", "8XS7-P53H-JKIV", now),
		key.NewKey("KEY-CS2-PRIME", "JPE6-MQV6-P7ST", now),
		key.NewKey("KEY-EFT", "SAPG-A2GR-0ULS", now),
		key.NewKey("KEY-GTA5", "T2DU-IJ1S-U16P", now),
		key.NewKey("KEY-CS2-PRIME", "WSSY-QTR7-Z57J", now),
		key.NewKey("KEY-GTA5", "U74E-EPCI-CY26", now),
		key.NewKey("KEY-CS2-PRIME", "FZXF-58H8-OR93", now),
		key.NewKey("KEY-EFT", "FPSM-HLZA-TPAL", now),
		key.NewKey("KEY-GTA5", "WSC9-28DJ-B2JE", now),
		key.NewKey("KEY-EFT", "P63J-F7UZ-DCYP", now),
		key.NewKey("KEY-CS2-PRIME", "C7W2-D4C5-QMT7", now),
		key.NewKey("KEY-GTA5", "JESI-DFBH-LK1K", now),
		key.NewKey("KEY-GTA5", "SGMA-JA0T-GR7D", now),
		key.NewKey("KEY-EFT", "3PR4-OSY9-M3ZW", now),
		key.NewKey("KEY-CS2-PRIME", "OMBE-C0JF-D45Y", now),
		key.NewKey("KEY-EFT", "KIKQ-FQJ8-9TI8", now),
		key.NewKey("KEY-GTA5", "LMAN-RSHS-AJDO", now),
		key.NewKey("KEY-CS2-PRIME", "BAKI-VT1X-Z5OL", now),
		key.NewKey("KEY-EFT", "9F0X-B46W-03FS", now),
		key.NewKey("KEY-GTA5", "S423-V6YY-IBEM", now),
		key.NewKey("KEY-CS2-PRIME", "D4UW-WYRA-20ST", now),
		key.NewKey("KEY-EFT", "XC0J-CJ0H-09RN", now),
		key.NewKey("KEY-CS2-PRIME", "RY1W-XCFJ-0KUA", now),
		key.NewKey("KEY-GTA5", "CJYY-YKSQ-QE6H", now),
		key.NewKey("KEY-EFT", "97AQ-38QJ-H8HU", now),
		key.NewKey("KEY-CS2-PRIME", "FS8E-3S5Z-I6RA", now),
		key.NewKey("KEY-GTA5", "ARQK-FML4-A14E", now),
		key.NewKey("KEY-EFT", "7Z6K-NO9V-MPJB", now),
		key.NewKey("KEY-CS2-PRIME", "D4K7-IJSG-N853", now),
		key.NewKey("KEY-GTA5", "W67T-ZB0Q-1XKB", now),
		key.NewKey("KEY-EFT", "7EQM-K09J-XKUO", now),
	}

	if err := tx.WithTransaction(
		ctx,
		transactor.Options{
			IsolationLevel: transactor.IsolationLevelReadCommitted,
		},
		func(ctx context.Context) error {
			seedLog.Info("seeding products")

			for _, p := range products {
				if err := productRepo.Create(ctx, p); err != nil {
					switch {
					case errors.Is(err, product.ErrAlreadyExists):
						continue
					default:
						return fmt.Errorf("create key sku=%s: %w", p.SKU, err)
					}
				}
			}

			seedLog.Info("products seeded")
			seedLog.Info("seeding keys")

			for _, k := range keys {
				if _, err := keyRepo.Create(ctx, k); err != nil {
					fmt.Println(k, err)
					switch {
					case errors.Is(err, key.ErrAlreadyExists):
						continue
					default:
						return fmt.Errorf("create key value=%s: %w", k.Value, err)
					}
				}
			}

			seedLog.Info("keys seeded")

			return nil
		},
	); err != nil {
		return fmt.Errorf("tx: %w", err)
	}

	return nil
}
