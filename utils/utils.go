package utils

import (
	"fmt"
)

func PrintColor(color, s string) string {
	var colorCode string

	switch color {
	case "red":
			colorCode = "\033[31m"
	case "green":
			colorCode = "\033[32m"
	case "blue":
			colorCode = "\033[34m"
	case "yellow":
			colorCode = "\033[33m"
	default:
			colorCode = "\033[0m" // Default color
	}

	return fmt.Sprintf("%s%s\033[0m", colorCode, s)
}

// func transferETH(client *ethclient.Client, fromPrivKey *ecdsa.PrivateKey, to common.Address, amount *big.Int) error {
// 	ctx := context.Background()

// 	publicKey := fromPrivKey.Public()
// 	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
// 	if !ok {
// 		return fmt.Errorf("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
// 	}

// 	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

// 	nonce, err := client.PendingNonceAt(ctx, fromAddress)
// 	if err != nil {
// 		return err
// 	}

// 	gasLimit := uint64(21000) // in units

// 	gasPrice, err := client.SuggestGasPrice(ctx)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	tx := types.NewTransaction(nonce, to, amount, gasLimit, gasPrice, nil)

// 	chainID := big.NewInt(1337)

// 	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), fromPrivKey)
// 	if err != nil {
// 		return err
// 	}

// 	return client.SendTransaction(ctx, signedTx)
// }