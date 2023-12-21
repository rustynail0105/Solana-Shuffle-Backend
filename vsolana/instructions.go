package vsolana

import (
	"context"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/system"
	tokenprogram "github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/solanashuffle/backend/env"
)

func SendNFTInstructions(fromAccount solana.PrivateKey, toAccount solana.PublicKey, mint solana.PublicKey) ([]solana.Instruction, error) {
	rpcClient := rpc.New(env.GetRPCUrl())

	fromAccountAta, _, err := solana.FindAssociatedTokenAddress(fromAccount.PublicKey(), mint)
	if err != nil {
		return []solana.Instruction{}, err
	}

	toAccountAta, _, err := solana.FindAssociatedTokenAddress(toAccount, mint)
	if err != nil {
		return []solana.Instruction{}, err
	}

	resp, err := rpcClient.GetAccountInfoWithOpts(
		context.TODO(),
		toAccountAta,
		&rpc.GetAccountInfoOpts{
			Encoding: solana.EncodingJSONParsed,
		},
	)

	var instructions []solana.Instruction

	if err != nil || resp.Value == nil || !resp.Value.Owner.Equals(TokenProgramID) {
		instructions = append(instructions,
			associatedtokenaccount.NewCreateInstruction(
				env.House().PublicKey(),
				toAccount,
				mint,
			).Build(),
		)
	}
	instructions = append(instructions,
		tokenprogram.NewTransferInstruction(
			1,
			fromAccountAta,
			toAccountAta,
			fromAccount.PublicKey(),
			[]solana.PublicKey{},
		).Build(),
	)

	return instructions, nil
}

func CreateAccountAndSendTokenInstructions(fromAccount solana.PrivateKey, toAccount solana.PublicKey, token solana.PublicKey, amount int) ([]solana.Instruction, error) {
	if amount <= 0 {
		return []solana.Instruction{}, nil
	}

	rpcClient := rpc.New(env.GetRPCUrl())
	toAccountAta, _, err := solana.FindAssociatedTokenAddress(toAccount, token)
	if err != nil {
		return []solana.Instruction{}, err
	}
	fromAccountAta, _, err := solana.FindAssociatedTokenAddress(fromAccount.PublicKey(), token)
	if err != nil {
		return []solana.Instruction{}, err
	}

	instructions := []solana.Instruction{}

	_, err = rpcClient.GetAccountInfoWithOpts(
		context.TODO(),
		toAccountAta,
		&rpc.GetAccountInfoOpts{
			Encoding: solana.EncodingJSONParsed,
		},
	)
	if err != nil {
		instructions = append(instructions,
			associatedtokenaccount.NewCreateInstruction(
				env.House().PublicKey(),
				toAccount,
				token,
			).Build(),
		)
	}
	instructions = append(instructions,
		tokenprogram.NewTransferInstruction(
			uint64(amount),
			fromAccountAta,
			toAccountAta,
			fromAccount.PublicKey(),
			[]solana.PublicKey{},
		).Build(),
	)

	return instructions, nil
}

func SendSOLInstructions(fromAccount solana.PrivateKey, toAccount solana.PublicKey, lamports int) []solana.Instruction {
	return []solana.Instruction{
		system.NewTransferInstruction(
			uint64(lamports),
			fromAccount.PublicKey(),
			toAccount,
		).Build(),
	}
}
