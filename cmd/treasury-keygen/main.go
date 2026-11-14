// treasury-keygen creates the Callit treasury wallet and stores its secret key
// in .env (mode 0600). It prints ONLY the public address, so the secret never
// appears in a terminal, a log or a chat. Safe to re-run: it never overwrites
// an existing key.
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gagliardetto/solana-go"
)

func main() {
	path := ".env"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	raw, _ := os.ReadFile(path)
	for _, line := range strings.Split(string(raw), "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "TREASURY_SECRET_KEY="); ok && strings.TrimSpace(v) != "" {
			k, err := solana.PrivateKeyFromBase58(strings.TrimSpace(v))
			if err != nil {
				log.Fatal("existing TREASURY_SECRET_KEY is not valid base58")
			}
			fmt.Println("Treasury already exists. Address:", k.PublicKey())
			return
		}
	}
	w := solana.NewWallet()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if len(raw) > 0 && raw[len(raw)-1] != '\n' {
		_, _ = f.WriteString("\n")
	}
	if _, err := fmt.Fprintf(f, "TREASURY_SECRET_KEY=%s\nPAYOUT_MAX_USDC=100\n", w.PrivateKey.String()); err != nil {
		log.Fatal(err)
	}
	_ = os.Chmod(path, 0o600)
	fmt.Println("Treasury created. Fund this address with USDC and a little SOL:", w.PublicKey())
}
