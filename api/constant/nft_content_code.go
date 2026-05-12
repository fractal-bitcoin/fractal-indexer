package constant

import "strings"

// NFTContentCodeContents maps ClickHouse content_code 1..255 to raw NFT
// content. Code 0 is reserved and means the row stores the original value in
// the content column.
//
// Keep the order stable once data is written: changing an existing entry
// changes the meaning of historical content_code values.
//
// Fill NFTContentCodeContentsText from the high-frequency content list, one
// content per line. Paste the content.txt body directly between the backticks.
// Example:
//
//	{"p":"brc-20","op":"mint","tick":"ordi","amt":"1000"}
//	{"p":"brc-20","op":"mint","tick":"sats","amt":"1000"}
const NFTContentCodeContentsText = `{"p":"brc-20","op":"mint","tick":"Nikola","amt":"3693"}
{"p":"brc-20","op":"mint","tick":"BigFloppa","amt":"100"}
{"p":"brc-20","op":"mint","tick":"hyenas","amt":"21000000"}
{"p":"brc-20","op":"mint","tick":"Pi_Token","amt":"1000"}
{"p":"brc-20","op":"mint","tick":"Piin_Token","amt":"1000"}
{"p":"brc-20","op":"mint","tick":"Sunbees","amt":"1000000"}
{"p":"brc-20","op":"mint","tick":"moondogs","amt":"10000000"}
{"p":"brc-20","op":"mint","tick":"Shibes","amt":"21000000"}
{"p":"brc-20","op":"mint","tick":"AI_GROK","amt":"4269"}
{"p":"brc-20","op":"mint","tick":"MoonRats","amt":"1000000"}
{"p":"brc-20","op":"mint","tick":"MoonCats","amt":"1000"}
{"p":"brc-20","op":"mint","tick":"MoonYetis","amt":"1000"}
{"p":"brc-20","op":"mint","tick":"MoonMother","amt":"1000000"}
{"p":"brc-20","op":"mint","tick":"MoonAnts","amt":"1000000"}
{"p":"brc-20","op":"mint","tick":"ProcessingX","amt":"4.2"}
{"p":"brc-20","op":"mint","tick":"MiniPotato","amt":"10000000"}
{"p":"brc-20","op":"mint","tick":"PiCode","amt":"9999"}
{"p":"brc-20","op":"mint","tick":"RippleX","amt":"1000000"}
{"p": "brc-20", "op": "mint", "tick": "MOLTBOOK", "amt": "2100"}
{"p":"brc-20","op":"mint","tick":"ETHERS","amt":"4444"}
{"p": "brc-20", "op": "mint", "tick": "Nikola", "amt": "3693"}
{"p":"brc-20","op":"mint","tick":"Recursive","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"dogecoin","amt":"1"}
{"p":"brc-20","op":"mint","tick":"gospel","amt":"100000000"}
{"p":"brc-20","op":"mint","tick":"MoonHonk","amt":"100000000"}
{"p":"brc-20","op":"mint","tick":"FatElvis","amt":"1000"}
{"p":"brc-20","op":"mint","tick":"cutepepe","amt":"100000000"}
{"p":"brc-20","op":"mint","tick":"people","amt":"1"}
{"p":"brc-20","op":"mint","tick":"pizzetta","amt":"100"}
{"p":"brc-20","op":"mint","tick":"BitLen","amt":"1"}
{"p":"brc-20","op":"mint","tick":"viperr","amt":"1"}
{"p":"brc-20","op":"mint","tick":"Moltbook","amt":"2100"}
{"p":"brc-20","op":"mint","tick":"minisats","amt":"100000000"}
{"p":"brc-20","op":"mint","tick":"AIFINK","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"HappyBeans","amt":"10000"}
{"p":"brc-20","op":"mint","tick":"LifeIsGood","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"Farted","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"Cocoro","amt":"1000"}
{"p":"brc-20","op":"mint","tick":"MiniMask","amt":"1000000"}
{"p":"brc-20","op":"mint","tick":"AppleInc","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"PROCESSINGX","amt":"4"}
{"p":"brc-20","op":"mint","tick":"undefined","amt":"1"}
{"p": "brc-20", "op": "mint", "tick": "BigFloppa", "amt": "100"}
{"p": "brc-20", "op": "mint", "tick": "PIPUNK", "amt": "9999"}
{"p": "brc-20", "op": "mint", "tick": "MoonAnts", "amt": "1000000"}
{"p":"brc-20","op":"mint","tick":"FBhexa","amt":"100000"}
{"p":"brc-20","op":"mint","tick":"ishowmeat","amt":"1"}
{"p": "brc-20", "op": "mint", "tick": "ITCOINS", "amt": "1111"}
{"p":"brc-20","op":"mint","tick":"naperdoto","amt":"1"}
{"p":"brc-20","op":"mint","tick":"MoonFractal","amt":"1000"}
{"p":"brc-20","op":"mint","tick":"TOKEN2049","amt":"100000000"}
{"p": "brc-20", "op": "mint", "tick": "RIPPLEX", "amt": "1000000"}
{"p":"brc-20","op":"mint","tick":"FIFA1930","amt":"3"}
{"p":"brc-20","op":"mint","tick":"Picat_Token","amt":"1000"}
{"p":"brc-20","op":"mint","tick":"MagnetDog","amt":"100000"}
{"p":"brc-20","op":"mint","tick":"Pijojo","amt":"68880"}
{"p":"brc-20","op":"mint","tick":"DOGgoMOON","amt":"10000"}
{"p":"brc-20","op":"mint","tick":"AVAVFB","amt":"69"}
{"p":"brc-20","op":"mint","tick":"FR4CTAL","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"WOJTEK","amt":"19421942"}
{"p":"brc-20","op":"mint","tick":"PICKLE","amt":"42069"}
{"p":"brc-20","op":"mint","tick":"Moonlights","amt":"10000000"}
{"p":"brc-20","op":"mint","tick":"MiniPizza","amt":"21000000"}
{"p":"brc-20","op":"mint","tick":"Moonchild","amt":"10000000"}
{"p":"brc-20","op":"mint","tick":"Maizes","amt":"816200"}
{"p":"brc-20","op":"mint","tick":"Meme_TRUMP","amt":"1"}
{"p":"brc-20","op":"mint","tick":"RT1doubleRR","amt":"1000000"}
{"p":"brc-20","op":"mint","tick":"RedBook","amt":"100000"}
{"p":"brc-20","op":"mint","tick":"Huluwas","amt":"1"}
{"p":"brc-20","op":"mint","tick":"DBALLZ","amt":"29129129.1"}
{"p": "brc-20", "op": "mint", "tick": "GREENBITCOIN", "amt": "4444"}
{"p":"brc-20","op":"mint","tick":"GLIZZY","amt":"42042.042042"}
{"p":"brc-20","op":"mint","tick":"TO_THE_MOON","amt":"10000"}
{"p":"brc-20","op":"mint","tick":"MOONMOTHER","amt":"1000000"}
{"p":"brc-20","op":"mint","tick":"RT1tutSUKA","amt":"1000000"}
{"p":"brc-20","op":"mint","tick":"TRUMP_II","amt":"1"}
{"p":"brc-20","op":"mint","tick":"chicken","amt":"3"}
{"p":"brc-20","op":"mint","tick":"PiPunk","amt":"9999"}
{"p": "brc-20", "op": "mint", "tick": "PROCESSINGX", "amt": "4"}
{"p":"brc-20","op":"mint","tick":"PotatoGame","amt":"888888"}
{"p":"brc-20","op":"mint","tick":"Angeldoges","amt":"2928"}
{"p":"brc-20","op":"mint","tick":"DicePlay","amt":"123456"}
{"p": "brc-20", "op": "mint", "tick": "AppleInc", "amt": "4444"}
{"p": "brc-20", "op": "mint", "tick": "Sunbees", "amt": "1000000"}
{"p":"brc-20","op":"mint","tick":"PICKLE","amt":"42069.42069"}
{"p":"brc-20","op":"mint","tick":"mooncats","amt":"1000"}
{"p":"brc-20","op":"mint","tick":"MOLTBOOK","amt":"2100"}
{"p":"brc-20","op":"mint","tick":"MoonBats","amt":"1000000"}
{"p":"brc-20","op":"mint","tick":"LiquidX","amt":"10000"}
{"p":"brc-20","op":"mint","tick":"skypig","amt":"99999999"}
{"p": "brc-20", "op": "mint", "tick": "MoonYetis", "amt": "1000"}
{"p":"brc-20","op":"mint","tick":"elizaAi16Z","amt":"200"}
{"p":"brc-20","op":"mint","tick":"cherry","amt":"5"}
{"p":"brc-20","op":"mint","tick":"MoonIn","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"DOGECORE","amt":"420420"}
{"p": "brc-20", "op": "mint", "tick": "Farted", "amt": "4444"}
{"p": "brc-20", "op": "mint", "tick": "SHUTDOWN", "amt": "4444"}
{"p":"brc-20","op":"mint","tick":"SCARFACE","amt":"42042"}
{"p":"brc-20","op":"mint","tick":"Pidoge","amt":"88888"}
{"p":"brc-20","op":"mint","tick":"JOKERS","amt":"72843.726"}
{"p":"brc-20","op":"mint","tick":"rabbit","amt":"5"}
{"p":"brc-20","op":"mint","tick":"Terminus","amt":"5"}
{"p":"brc-20","op":"mint","tick":"potatopizza","amt":"5"}
{"p":"brc-20","op":"mint","tick":"sandwich","amt":"5"}
{"p": "brc-20", "op": "mint", "tick": "LifeIsGood", "amt": "4444"}
{"p":"brc-20","op":"mint","tick":"banana","amt":"5"}
{"p": "brc-20", "op": "mint", "tick": "AIFINK", "amt": "4444"}
{"p":"brc-20","op":"mint","tick":"TRUMPF1GHT","amt":"4747"}
{"p":"brc-20","op":"mint","tick":"coffee","amt":"5"}
{"p":"brc-20","op":"mint","tick":"GreenBitcoin","amt":"4444"}
{"p": "brc-20", "op": "mint", "tick": "FBHEXA", "amt": "100000"}
{"p":"brc-20","op":"mint","tick":"elephant","amt":"5"}
{"p":"brc-20","op":"mint","tick":"FENNEC","amt":"5"}
{"p":"brc-20","op":"mint","tick":"Moonfoxss","amt":"10000000"}
{"p":"brc-20","op":"mint","tick":"FB_BAN","amt":"100"}
{"p": "brc-20", "op": "mint", "tick": "MOONIN", "amt": "4444"}
{"p":"brc-20","op":"mint","tick":"RoyalMadrd","amt":"1"}
{"p":"brc-20","op":"mint","tick":"GooGee","amt":"1"}
{"p": "brc-20", "op": "mint", "tick": "RECURSIONS", "amt": "4444"}
{"p":"brc-20","op":"mint","tick":"turtle","amt":"6"}
{"p":"brc-20","op":"mint","tick":"ManUtdFC","amt":"0.009"}
{"p": "brc-20", "op": "mint", "tick": "Cocoro", "amt": "1000"}
{"p":"brc-20","op":"mint","tick":"bitmap","amt":"6"}
{"p":"brc-20","op":"mint","tick":"helloworld","amt":"99"}
{"p":"brc-20","op":"mint","tick":"moonrats","amt":"1000000"}
{"p":"brc-20","op":"mint","tick":"bitman","amt":"6"}
{"p":"brc-20","op":"mint","tick":"MakerDAO","amt":"1000000"}
{"p":"brc-20","op":"mint","tick":"OfuckO","amt":"1"}
{"p": "brc-20", "op": "mint", "tick": "MiniPotato", "amt": "10000000"}
{"p":"brc-20","op":"mint","tick":"LuckySW","amt":"1"}
{"p":"brc-20","op":"mint","tick":"Solana","amt":"490000000"}
{"p":"brc-20","op":"mint","tick":"DOGEFIFA","amt":"1"}
{"p":"brc-20","op":"mint","tick":"EURO2028","amt":"1"}
{"p":"brc-20","op":"mint","tick":"GoldenSnakes","amt":"1"}
{"p":"brc-20","op":"mint","tick":"dolphin","amt":"6"}
{"p":"brc-20","op":"mint","tick":"FIFA2026","amt":"1"}
{"p": "brc-20", "op": "mint", "tick": "MiniMask", "amt": "1000000"}
{"p":"brc-20","op":"mint","tick":"nikola","amt":"3693"}
{"p":"brc-20","op":"mint","tick":"MoonGods","amt":"5"}
{"p":"brc-20","op":"mint","tick":"TRUMPstrump","amt":"333"}
{"p": "brc-20", "op": "mint", "tick": "TOKEN2049", "amt": "100000000"}
{"p":"brc-20","op":"mint","tick":"AcMilanFC","amt":"0.1"}
{"p":"brc-20","op":"mint","tick":"Nikola","amt":"3639"}
{"p":"brc-20","op":"mint","tick":"OG1930","amt":"1"}
{"p":"brc-20","op":"mint","tick":"PiCowCoin","amt":"6283.19"}
{"p":"brc-20","op":"mint","tick":"octopus","amt":"6"}
{"p": "brc-20", "op": "mint", "tick": "SCARFACE", "amt": "42042.042042"}
{"p": "brc-20", "op": "mint", "tick": "minisats", "amt": "100000000"}
{"p":"brc-20","op":"mint","tick":"PieDog","amt":"9999"}
{"p":"brc-20","op":"mint","tick":"LuckyPi","amt":"1"}
{"p": "brc-20", "op": "mint", "tick": "BIGFLOPPA", "amt": "100"}
{"p": "brc-20", "op": "mint", "tick": "NIKOLA", "amt": "3693"}
{"p":"brc-20","op":"mint","tick":"TRUMPCTO","amt":"333"}
{"p": "brc-20", "op": "mint", "tick": "HappyBeans", "amt": "10000"}
{"p":"brc-20","op":"mint","tick":"FB10000XSATS","amt":"50000"}
{"p":"brc-20","op":"mint","tick":"kitten","amt":"6"}
{"p":"brc-20","op":"mint","tick":"SCARFACE","amt":"42042.042042"}
{"p":"brc-20","op":"mint","tick":"turkey","amt":"6"}
{"p":"brc-20","op":"mint","tick":"1_3BTC","amt":"1"}
{"p":"brc-20","op":"mint","tick":"SHUTDOWN","amt":"4444"}
{"p": "brc-20", "op": "mint", "tick": "RedBook", "amt": "100000"}
{"p":"brc-20","op":"mint","tick":"cutebeat","amt":"10000"}
{"p":"brc-20","op":"mint","tick":"CommonSense","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"PIPUNK","amt":"9999"}
{"p":"brc-20","op":"mint","tick":"maizes","amt":"816200"}
{"p":"brc-20","op":"mint","tick":"MEGATRUMP","amt":"10"}
{"p":"brc-20","op":"mint","tick":"onzerol","amt":"1000"}
{"p":"brc-20","op":"mint","tick":"USDONE","amt":"1000"}
{"p": "brc-20", "op": "mint", "tick": "TO_THE_MOON", "amt": "10000"}
{"p":"brc-20","op":"mint","tick":"Tators","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"GLIZZY","amt":"42042"}
{"p":"brc-20","op":"mint","tick":"PresleyElvis","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"potato","amt":"10"}
{"p":"brc-20","op":"mint","tick":"PiCowCoin","amt":"6283"}
{"p":"brc-20","op":"mint","tick":"MoonFrogs","amt":"1000"}
{"p":"brc-20","op":"mint","tick":"Wedges","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"Oranges","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"PROCESSINGX","amt":"4.2"}
{"p":"brc-20","op":"mint","tick":"orange","amt":"100"}
{"p":"brc-20","op":"mint","tick":"Porsha","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"BenAffleck","amt":"4443"}
{"p":"brc-20","op":"mint","tick":"NVIDIACorp","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"OprahWinfrey","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"pandora","amt":"1"}
{"p":"brc-20","op":"mint","tick":"FireCoin","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"AIELON","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"cornbun","amt":"10000"}
{"p":"brc-20","op":"mint","tick":"TRUMPCEO","amt":"3.33"}
{"p":"brc-20","op":"mint","tick":"mushroom","amt":"10"}
{"p":"brc-20","op":"mint","tick":"EOLNMUSK","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"tomato","amt":"100"}
{"p":"brc-20","op":"mint","tick":"FBSATOSHIBTC","amt":"99"}
{"p":"brc-20","op":"mint","tick":"Peanutbutter","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"beefsteak","amt":"100000"}
{"p":"brc-20","op":"mint","tick":"GALACTIC","amt":"1000"}
{"p":"brc-20","op":"mint","tick":"TrumpMoney","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"pepepe","amt":"1000"}
{"p":"brc-20","op":"mint","tick":"ElonMusks","amt":"4442"}
{"p":"brc-20","op":"mint","tick":"MoonFIST","amt":"1"}
{"p": "brc-20", "op": "mint", "tick": "WIFCOIN", "amt": "1111"}
{"p":"brc-20","op":"mint","tick":"Doughnuts","amt":"4444"}
{"p": "brc-20", "op": "mint", "tick": "POTATOGAME", "amt": "888888"}
{"p":"brc-20","op":"mint","tick":"SPENCER","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"spider","amt":"6"}
{"p":"brc-20","op":"mint","tick":"cutefrog","amt":"4442"}
{"p":"brc-20","op":"mint","tick":"Fruits","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"oogler","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"doodles","amt":"5"}
{"p":"brc-20","op":"mint","tick":"MoonWif","amt":"1000000000000"}
{"p":"brc-20","op":"mint","tick":"BoonDog","amt":"3333"}
{"p":"brc-20","op":"mint","tick":"MattDamon","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"ELFCAT","amt":"10"}
{"p":"brc-20","op":"mint","tick":"watermelon","amt":"10"}
{"p":"brc-20","op":"mint","tick":"100Bitcoin","amt":"1001"}
{"p":"brc-20","op":"mint","tick":"Taiwan","amt":"10"}
{"p":"brc-20","op":"mint","tick":"blueberry","amt":"1000"}
{"p":"brc-20","op":"mint","tick":"ElonAi","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"AITRUMP","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"DBALLZ","amt":"29129129"}
{"p":"brc-20","op":"mint","tick":"cooldog","amt":"10"}
{"p":"brc-20","op":"mint","tick":"BnPizza","amt":"10"}
{"p":"brc-20","op":"mint","tick":"Mooncity","amt":"1000"}
{"p":"brc-20","op":"mint","tick":"BeLife","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"PieSat","amt":"314314"}
{"p":"brc-20","op":"mint","tick":"bigban","amt":"10"}
{"p":"brc-20","op":"mint","tick":"LadyBoner","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"Circles","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"PEPE_FRACTAL","amt":"200000000"}
{"p":"brc-20","op":"mint","tick":"Cadence","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"WestCOL","amt":"1"}
{"p":"brc-20","op":"mint","tick":"PooCoin","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"itcoins","amt":"1111"}
{"p":"brc-20","op":"mint","tick":"PiPunk","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"Toshis","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"bamboo","amt":"10"}
{"p":"brc-20","op":"mint","tick":"ai_grok","amt":"4269"}
{"p":"brc-20","op":"mint","tick":"fiba2026","amt":"2"}
{"p":"brc-20","op":"mint","tick":"FBHEXA","amt":"100000"}
{"p":"brc-20","op":"mint","tick":"BitcoinPlus","amt":"20000"}
{"p": "brc-20", "op": "mint", "tick": "spider", "amt": "6"}
{"p":"brc-20","op":"mint","tick":"mango","amt":"500"}
{"p":"brc-20","op":"mint","tick":"moonants","amt":"1000000"}
{"p":"brc-20","op":"mint","tick":"888888","amt":"100"}
{"p": "brc-20", "op": "mint", "tick": "HOPECOIN", "amt": "4444"}
{"p": "brc-20", "op": "mint", "tick": "BITCOINPLUS", "amt": "20000"}
{"p":"brc-20","op":"mint","tick":"mooncats","amt":"999"}
{"p":"brc-20","op":"mint","tick":"blackdoge","amt":"10"}
{"p":"brc-20","op":"mint","tick":"PotatoHash","amt":"10"}
{"p":"brc-20","op":"mint","tick":"FBitcoin","amt":"100"}
{"p":"brc-20","op":"mint","tick":"StephCurry","amt":"3030"}
{"p":"brc-20","op":"mint","tick":"SendMoney","amt":"4444"}
{"p":"brc-20","op":"mint","tick":"fbrc20_sats","amt":"100000000"}
{"p": "brc-20", "op": "mint", "tick": "MOONYETIS", "amt": "1000"}
{"p":"brc-20","op":"mint","tick":"AlexHormozi","amt":"4444"}
`

var NFTContentCodeContents = parseNFTContentCodeContents(NFTContentCodeContentsText)

var nftContentCodeByContent map[string]uint8

func init() {
	if len(NFTContentCodeContents) > 255 {
		panic("constant.NFTContentCodeContents supports at most 255 entries")
	}

	nftContentCodeByContent = make(map[string]uint8, len(NFTContentCodeContents))
	for i, content := range NFTContentCodeContents {
		if content == "" {
			panic("constant.NFTContentCodeContents cannot contain empty content")
		}
		if _, ok := nftContentCodeByContent[content]; ok {
			panic("constant.NFTContentCodeContents cannot contain duplicate content")
		}
		nftContentCodeByContent[content] = uint8(i + 1)
	}
}

func parseNFTContentCodeContents(text string) []string {
	text = strings.TrimPrefix(text, "\n")
	text = strings.TrimSuffix(text, "\n")
	text = strings.TrimSuffix(text, "\r")
	if text == "" {
		return nil
	}

	lines := strings.Split(text, "\n")
	contents := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			continue
		}
		contents = append(contents, line)
	}
	return contents
}

// GetNFTContentCode returns 0 when content should be stored directly in the
// ClickHouse content column.
func GetNFTContentCode(content []byte) uint8 {
	if len(content) == 0 || len(nftContentCodeByContent) == 0 {
		return 0
	}
	return nftContentCodeByContent[string(content)]
}

// GetNFTContentByCode returns the raw NFT content for a non-zero content_code.
func GetNFTContentByCode(code uint8) (string, bool) {
	if code == 0 || int(code) > len(NFTContentCodeContents) {
		return "", false
	}
	return NFTContentCodeContents[int(code)-1], true
}

// EncodeNFTContentForDB returns the content_code and the value to write to the
// ClickHouse content column. When code is non-zero, dbContent is nil/empty.
func EncodeNFTContentForDB(content []byte) (code uint8, dbContent []byte) {
	code = GetNFTContentCode(content)
	if code != 0 {
		return code, nil
	}
	return 0, content
}
