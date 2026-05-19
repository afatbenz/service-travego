package config

import (
	"os"

	"github.com/veritrans/go-midtrans"
)

// MidtransConfig menyimpan konfigurasi Midtrans
type MidtransConfig struct {
	Client midtrans.Client
	Snap   midtrans.SnapGateway
}

// InitMidtrans menginisialisasi client Midtrans
func InitMidtrans() *MidtransConfig {
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	clientKey := os.Getenv("MIDTRANS_CLIENT_KEY")
	env := os.Getenv("MIDTRANS_ENV")

	midtransEnv := midtrans.Sandbox
	if env == "production" {
		midtransEnv = midtrans.Production
	}

	client := midtrans.NewClient()
	client.ServerKey = serverKey
	client.ClientKey = clientKey
	client.APIEnvType = midtransEnv

	snap := midtrans.SnapGateway{
		Client: client,
	}

	return &MidtransConfig{
		Client: client,
		Snap:   snap,
	}
}
