package shuffle

/*

var (
	intermediaryDelay = time.Millisecond * 333
)

func (u *SessionUser) ParseIntermediary(account solana.PublicKey, token Token, iterations int, commitment rpc.CommitmentType) (Assets, error) {
	if token.PublicKey != solana.SolMint {
		var beforeAmount int
		for _, asset := range u.Assets {
			if !asset.IsToken() || asset.Mint != token.PublicKey {
				continue
			}
			beforeAmount += asset.Value()
		}

		assets, err := parseIntermediarySPL(account, token, iterations, beforeAmount, commitment)
		if err != nil {
			return Assets{}, err
		}

		newAssets := Assets{
			{
				Type:  "Token",
				Mint:  token.PublicKey,
				Price: assets.Value() - beforeAmount,
			},
		}

		u.Assets = assets
		u.Value = u.Assets.Value()

		return newAssets, nil
	}

	fmt.Println(u.Assets)
	assets, err := parseIntermediarySOLOrNFTs(account, iterations, u.Assets, commitment)
	if err != nil {
		return Assets{}, err
	}

	newAssets := Assets{}
	for _, asset := range assets {
		if asset.IsToken() {
			for _, beforeAsset := range u.Assets {
				if !beforeAsset.IsToken() {
					continue
				}

				newAmount := asset.Value() - beforeAsset.Value()
				if newAmount == 0 {
					continue
				}

				newAssets = append(newAssets, GeneralAsset{
					Type:  "Token",
					Mint:  solana.SolMint,
					Price: newAmount,
				})
			}
		} else if asset.IsNFT() {
			var found bool
			for _, beforeAsset := range u.Assets {
				if beforeAsset.Mint == asset.Mint {
					found = true
				}
			}
			if !found {
				newAssets = append(newAssets, asset)
			}
		}

	}

	fmt.Println(assets)

	u.Assets = assets
	u.Value = u.Assets.Value()

	return newAssets, nil
}

func parseIntermediarySOLOrNFTs(publicKey solana.PublicKey, iterations int, beforeAsset Assets, commitment rpc.CommitmentType) (Assets, error) {
	rpcClient := rpc.New(env.GetRPCUrl())

	var beforeNFTAssets Assets
	var beforeTokenAssets Assets
	for _, asset := range beforeAsset {
		if asset.IsNFT() {
			beforeNFTAssets = append(beforeNFTAssets, asset)
		} else {
			beforeTokenAssets = append(beforeTokenAssets, asset)
		}
	}

	for i := 0; i < iterations; i++ {
		if i > 0 {
			time.Sleep(intermediaryDelay)
		}

		var solAssets Assets
		var nftAssets Assets
		var outerErr error

		var wg sync.WaitGroup
		wg.Add(2)

		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			var err error
			solAssets, err = parseSOLTask(
				rpcClient,
				publicKey,
				commitment,
			)
			if err != nil {
				outerErr = err
			}
			fmt.Println(solAssets, err)
		}(&wg)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			var err error
			nftAssets, err = parseNFTsTask(
				rpcClient,
				publicKey,
				commitment,
			)
			if err != nil {
				outerErr = err
			}
			fmt.Println(nftAssets, err)
		}(&wg)

		wg.Wait()

		if outerErr != nil {
			continue
		}

		assets := nftAssets
		if solAssets.Value() > 0 {
			assets = append(assets, solAssets...)
		}

		if solAssets.Value() > beforeTokenAssets.Value() {
			fmt.Println("sol value", solAssets.Value(), beforeTokenAssets.Value())
			return assets, nil
		}

		if len(nftAssets) > len(beforeNFTAssets) {
			fmt.Println("nft length")
			return assets, nil
		}

		if func() bool {
			for _, asset := range nftAssets {
				var found bool
				for _, beforeAsset := range beforeNFTAssets {
					if asset.Mint == beforeAsset.Mint {
						found = true
					}
				}

				if !found {
					return true
				}
			}
			return false
		}() {
			fmt.Println("nft mints")
			return assets, nil
		}
	}

	return Assets{}, errors.New("no new assets found after timeout")
}

func parseIntermediarySOL(publicKey solana.PublicKey, iterations int, beforeAmount int, commitment rpc.CommitmentType) (Assets, error) {
	rpcClient := rpc.New(env.GetRPCUrl())

	for i := 0; i < iterations; i++ {
		if i > 0 {
			time.Sleep(intermediaryDelay)
		}

		assets, err := parseSOLTask(rpcClient, publicKey, commitment)
		if err != nil {
			continue
		}

		if assets.Value() <= beforeAmount {
			continue
		}

		return assets, nil
	}

	return Assets{}, errors.New("no new assets found after timeout")
}

func parseSOLTask(rpcClient *rpc.Client, publicKey solana.PublicKey, commitment rpc.CommitmentType) (Assets, error) {
	resp, err := rpcClient.GetBalance(
		context.TODO(),
		publicKey,
		commitment,
	)
	if err != nil {
		return Assets{}, err
	}

	amount := int(resp.Value)

	return Assets{
		{
			Type:  "Token",
			Price: amount,
			Mint:  solana.SolMint,
		},
	}, nil
}

func parseIntermediaryNFT(publicKey solana.PublicKey, iterations int, beforeNFTs Assets, commitment rpc.CommitmentType) (Assets, error) {
	rpcClient := rpc.New(env.GetRPCUrl())

	for i := 0; i < iterations; i++ {
		if i > 0 {
			time.Sleep(intermediaryDelay)
		}
		assets, err := parseNFTsTask(rpcClient, publicKey, commitment)
		if err != nil {
			continue
		}

		// was a new asset found?
		if len(assets) == len(beforeNFTs) {
			continue
		}

		if !func() bool {
			for _, asset := range assets {
				var found bool
				for _, beforeAsset := range beforeNFTs {
					if asset.Mint == beforeAsset.Mint {
						found = true
					}
				}

				if !found {
					return true
				}
			}
			return false
		}() {
			continue
		}

		return assets, nil
	}

	return Assets{}, errors.New("no new assets found after timeout")
}

func parseNFTsTask(rpcClient *rpc.Client, publicKey solana.PublicKey, commitment rpc.CommitmentType) (Assets, error) {
	resp, err := rpcClient.GetTokenAccountsByOwner(
		context.TODO(),
		publicKey,
		&rpc.GetTokenAccountsConfig{
			ProgramId: &solana.TokenProgramID,
		},
		&rpc.GetTokenAccountsOpts{
			Encoding:   solana.EncodingJSONParsed,
			Commitment: commitment,
		},
	)
	if err != nil {
		return Assets{}, err
	}
	metaAccounts := []solana.PublicKey{}
	for _, tokenAccount := range resp.Value {
		var tokenAccountData csolana.TokenAccountData
		err = json.Unmarshal(tokenAccount.Account.Data.GetRawJSON(), &tokenAccountData)
		if err != nil {
			return Assets{}, err
		}
		if tokenAccountData.Parsed.Info.Tokenamount.Decimals != 0 || tokenAccountData.Parsed.Info.Tokenamount.Amount != "1" {
			continue
		}
		metaAccount, _, err := solana.FindTokenMetadataAddress(tokenAccountData.Parsed.Info.Mint)
		if err != nil {
			continue
		}
		metaAccounts = append(metaAccounts, metaAccount)
	}

	accountChunks := utility.ChunkBy(metaAccounts, csolana.RPCMaximum)
	responseValues := make([]*rpc.Account, len(metaAccounts))
	var wg sync.WaitGroup
	wg.Add(len(accountChunks))

	for i, chunk := range accountChunks {
		go func(gi int, accounts []solana.PublicKey, wg *sync.WaitGroup) {
			defer wg.Done()
			resp, err := rpcClient.GetMultipleAccountsWithOpts(
				context.TODO(),
				accounts,
				&rpc.GetMultipleAccountsOpts{
					Encoding: solana.EncodingJSONParsed,
				},
			)
			if err != nil {
				return
			}

			for i, value := range resp.Value {
				responseValues[gi*csolana.RPCMaximum+i] = value
			}
		}(i, chunk, &wg)
	}
	wg.Wait()

	if len(responseValues) != len(metaAccounts) {
		return Assets{}, errors.New("rpc failed")
	}

	assets := Assets{}
	for _, account := range responseValues {
		if account == nil || account.Data == nil {
			continue
		}
		if len(account.Data.GetBinary()) != csolana.MetadataAccountSize {
			continue
		}
		tokenMetadata, err := csolana.DeserializeMetadata(account.Data.GetBinary())
		if err != nil {
			continue
		}

		asset := GeneralAsset{
			Type: "NFT",

			Mint:        tokenMetadata.Mint,
			MetadataURL: tokenMetadata.Data.Uri,
		}

		collectionSymbol, err := database.RDB.Get(
			context.TODO(),
			tokenMetadata.Mint.String(),
		).Result()
		if err != nil {
			continue
		}
		asset.CollectionSymbol = collectionSymbol

		hadeswapMarket, err := database.RDB.Get(
			context.TODO(),
			collectionSymbol,
		).Result()
		if err == nil {
			asset.HadeswapMarket, _ = solana.PublicKeyFromBase58(hadeswapMarket)
		}

		price, err := price.Estimate(collectionSymbol)
		if err != nil {
			continue
		}

		asset.Price = price

		assets = append(assets, asset)
	}

	return assets, nil
}

func parseIntermediarySPL(publicKey solana.PublicKey, token Token, iterations int, beforeAmount int, commitment rpc.CommitmentType) (Assets, error) {
	ata, _, err := solana.FindAssociatedTokenAddress(publicKey, token.PublicKey)
	if err != nil {
		return Assets{}, err
	}

	rpcClient := rpc.New(env.GetRPCUrl())

	for i := 0; i < iterations; i++ {
		if i > 0 {
			time.Sleep(intermediaryDelay)
		}
		resp, err := rpcClient.GetTokenAccountBalance(
			context.TODO(),
			ata,
			commitment,
		)
		if err != nil {
			continue
		}

		amount, err := strconv.Atoi(resp.Value.Amount)
		if err != nil {
			continue
		}

		if amount <= beforeAmount {
			continue
		}

		return Assets{
			{
				Type:  "Token",
				Price: amount,
				Mint:  token.PublicKey,
			},
		}, nil
	}

	return Assets{}, errors.New("no new assets found after timeout")
}

*/
