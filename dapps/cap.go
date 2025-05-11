package dapps

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/fatih/color"
	"github.com/joho/godotenv"
)

const (
	RPC_URL_MEGAETH          = "https://carrot.megaeth.com/rpc"
	CHAIN_ID_MEGAETH         = 6342
	EXPLORER_BASE_MEGAETH    = "https://www.megaexplorer.xyz/tx/"
	DELAY_SECONDS_MEGAETH    = 2
	CUSD_CONTRACT_ADDRESS    = "0xe9b6e75c243b6100ffcb1c66e8f78f96feea727f"
	GAS_LIMIT_BUFFER_PERCENT = 10
)

var (
	green   = color.New(color.FgGreen).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	cyan    = color.New(color.FgCyan).SprintFunc()
	magenta = color.New(color.FgMagenta).SprintFunc()
	blue    = color.New(color.FgBlue).SprintFunc()
)

type MintResultCap struct {
	Success     bool
	WalletIndex int
	TxHash      string
	Fee         string
	Error       error
}

func Cap() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("\nEnter number of mints: ")
	numInput, _ := reader.ReadString('\n')
	numInput = strings.TrimSpace(numInput)

	numMints, err := strconv.Atoi(numInput)
	if err != nil || numMints < 1 {
		fmt.Println(red("Invalid number. Please enter a positive integer."))
		os.Exit(1)
	}

	fmt.Printf("\nDApp: %s\n", cyan("CAP"))
	fmt.Printf("Mints: %s\n\n", magenta(numMints))

	CUSD(numMints)
}

func CUSD(numMints int) {
	godotenv.Load()

	wallets := make([]string, 20)
	for i := 0; i < 20; i++ {
		wallets[i] = os.Getenv(fmt.Sprintf("PRIVATE_KEYS_WALLET%d", i+1))
	}

	var activeWallets []string
	for i, key := range wallets {
		if key != "" {
			activeWallets = append(activeWallets, key)
			log.Printf("Loaded Wallet #%d", i+1)
		}
	}

	if len(activeWallets) == 0 {
		log.Fatal("No valid private keys found in environment variables")
	}

	client, err := ethclient.Dial(RPC_URL_MEGAETH)
	if err != nil {
		log.Fatalf("Failed to connect to MegaETH RPC: %v", err)
	}
	defer client.Close()

	cusdABI, err := getCUSDABI()
	if err != nil {
		log.Fatalf("ABI error: %v", err)
	}

	results := make(chan MintResultCap, numMints)
	var wg sync.WaitGroup

	walletMutexes := make([]sync.Mutex, len(activeWallets))

	for i := 0; i < numMints; i++ {
		wg.Add(1)
		walletIndex := i % len(activeWallets)

		go func(mintNum int, walletIdx int) {
			defer wg.Done()

			time.Sleep(time.Duration(mintNum*DELAY_SECONDS_MEGAETH) * time.Second)

			walletMutexes[walletIdx].Lock()
			defer walletMutexes[walletIdx].Unlock()

			results <- mintCUSD(activeWallets[walletIdx], walletIdx+1, cusdABI, client)
		}(i, walletIndex)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	successCount := 0
	for res := range results {
		if res.Success {
			successCount++
			fmt.Printf("[%s #%d]\n", cyan("Wallet"), res.WalletIndex)
			fmt.Printf("TxHash: %s\n", blue(EXPLORER_BASE_MEGAETH+res.TxHash))
			fmt.Printf("Fee: %s\n", res.Fee)

			client, err := ethclient.Dial(RPC_URL_MEGAETH)
			if err == nil {
				pk, err := crypto.HexToECDSA(strings.TrimPrefix(getPrivateKey(res.WalletIndex), "0x"))
				if err == nil {
					address := crypto.PubkeyToAddress(pk.PublicKey)
					cusdBalance, _ := getCUSDBalance(client, address, cusdABI)

					cusdBalanceFloat := new(big.Float).Quo(
						new(big.Float).SetInt(cusdBalance),
						big.NewFloat(1e18),
					)

					fmt.Printf("New cUSD Balance: %s cUSD\n\n", magenta(fmt.Sprintf("%.4f", cusdBalanceFloat)))
					fmt.Println("\n▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔")
				}
				client.Close()
			}
		} else {
			fmt.Printf("\n%s %s [%s #%d]\n",
				red("❌"), red("MINT FAILED"), cyan("Wallet"), res.WalletIndex)
			fmt.Printf("%s: %v\n", red("Error"), res.Error)
			fmt.Printf("%s: %s/%s\n", yellow("Total successfully minted"), green(successCount), magenta(numMints))
			return
		}
	}

	fmt.Printf("\n%s %s\n\n", green("✅"), green("MINT SUCCESS"))
	fmt.Println("Follow X : 0xNekowawolf\n")
	fmt.Printf("%s: %s/%s\n\n", yellow("Total successfully minted"), green(successCount), magenta(numMints))
}

func mintCUSD(privateKey string, walletIndex int, cusdABI abi.ABI, client *ethclient.Client) MintResultCap {
	pk, err := crypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
	if err != nil {
		return MintResultCap{Error: fmt.Errorf("invalid private key: %v", err)}
	}

	fromAddress := crypto.PubkeyToAddress(pk.PublicKey)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return MintResultCap{Error: fmt.Errorf("nonce error: %v", err)}
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return MintResultCap{Error: fmt.Errorf("gas price error: %v", err)}
	}

	contractAddress := common.HexToAddress(CUSD_CONTRACT_ADDRESS)
	amount := big.NewInt(1000)
	amount.Mul(amount, big.NewInt(1e18))

	data, err := cusdABI.Pack("mint", fromAddress, amount)
	if err != nil {
		return MintResultCap{Error: fmt.Errorf("ABI pack error: %v", err)}
	}

	gasLimit, err := estimateGasLimit(client, fromAddress, contractAddress, big.NewInt(0), data)
	if err != nil {
		return MintResultCap{Error: fmt.Errorf("gas estimate error: %v", err)}
	}

	tx := types.NewTransaction(
		nonce,
		contractAddress,
		big.NewInt(0),
		gasLimit,
		gasPrice,
		data,
	)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(CHAIN_ID_MEGAETH)), pk)
	if err != nil {
		return MintResultCap{Error: fmt.Errorf("signing error: %v", err)}
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return MintResultCap{Error: fmt.Errorf("send tx error: %v", err)}
	}

	receipt, err := bind.WaitMined(context.Background(), client, signedTx)
	if err != nil {
		return MintResultCap{Error: fmt.Errorf("tx mining error: %v", err)}
	}

	fee := new(big.Float).Quo(
		new(big.Float).SetInt(new(big.Int).Mul(big.NewInt(int64(receipt.GasUsed)), gasPrice)),
		new(big.Float).SetInt(big.NewInt(1e18)),
	)
	feeStr := strings.TrimRight(strings.TrimRight(fee.Text('f', 18), "0"), ".")

	return MintResultCap{
		Success:     true,
		WalletIndex: walletIndex,
		TxHash:      signedTx.Hash().Hex(),
		Fee:         yellow(feeStr + " ETH"),
	}
}

func getCUSDBalance(client *ethclient.Client, address common.Address, cusdABI abi.ABI) (*big.Int, error) {
	contractAddress := common.HexToAddress(CUSD_CONTRACT_ADDRESS)

	data, err := cusdABI.Pack("balanceOf", address)
	if err != nil {
		return nil, err
	}

	msg := ethereum.CallMsg{
		To:   &contractAddress,
		Data: data,
	}
	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return nil, err
	}

	var balance *big.Int
	err = cusdABI.UnpackIntoInterface(&balance, "balanceOf", result)
	if err != nil {
		return nil, err
	}

	return balance, nil
}

func getPrivateKey(walletIndex int) string {
	keys := getPrivateKeys()
	if walletIndex-1 < len(keys) {
		return keys[walletIndex-1]
	}
	return ""
}

func getPrivateKeys() []string {
	wallets := make([]string, 20)
	for i := 0; i < 20; i++ {
		wallets[i] = os.Getenv(fmt.Sprintf("PRIVATE_KEYS_WALLET%d", i+1))
	}

	var activeWallets []string
	for _, key := range wallets {
		if key != "" {
			activeWallets = append(activeWallets, key)
		}
	}
	return activeWallets
}

func estimateGasLimit(client *ethclient.Client, from common.Address, to common.Address, value *big.Int, data []byte) (uint64, error) {
	msg := ethereum.CallMsg{
		From:  from,
		To:    &to,
		Value: value,
		Data:  data,
	}
	gasLimit, err := client.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate gas: %v", err)
	}
	const GAS_LIMIT_BUFFER_PERCENT = 20
	gasLimitWithBuffer := gasLimit * (100 + GAS_LIMIT_BUFFER_PERCENT) / 100
	return gasLimitWithBuffer, nil
}

func getCUSDABI() (abi.ABI, error) {
	abiJSON := `[{"inputs":[{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"mint","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"account","type":"address"}],"name":"balanceOf","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}]`
	return abi.JSON(strings.NewReader(abiJSON))
}
