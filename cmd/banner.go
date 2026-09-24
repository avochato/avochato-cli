package cmd

import (
	"fmt"
	"os"
)

func colorsEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func printBanner() {
	green := ""
	reset := ""
	bold := ""

	if colorsEnabled() {
		green = "\033[32m"
		reset = "\033[0m"
		bold = "\033[1m\033[97m" // bold white for CLI text
	}

	fmt.Printf("%s", green)
	fmt.Print(`
                                                                     .fLLLLLLLLLLLLLLLLLLL
                                                               fLLLLLfiiiiiiiiiiiiiiiiiiiiLLL
                                                          1LLLL1iiiiiiiiiiiiiiiiiiiiiiiiiiiiLL;
                                                      ,LLLfiiiiiiiiii:...........;iiiiiiiiiifLL
                                                   ,LLLiiiiiiii;..................iiiiiiiii1LLL
                                                 LLLiiiiiiii,....................iiiiiiiiiLLLLL
`)
	fmt.Printf("                                               fLLiiiiiiii.........%sCLI%s........;iiiiiiii1LLLLLLL\n", bold+reset+green, reset+green)
	fmt.Print(`                                              LLiiiiiiiii,................;iiiiiiiii1LLLLLLLLL
                                             :Liiiiiiiiiiii;,....,:;iiiiiiiiiiiiiLLLLLLLLLLLL,
                                             1LLiiiiiiiiiiiiiiiiiiiiiiiiiiiiLLLLLLLLLLLLLLLL1
                                             tLLLL1iiiiiiiiiiiiiiiiiiiiiiLLLLLLLLLLLLLLLLLL;
                                             .LLLLLLLLLLLLLLLLLLiiiiiiiiLLLLLLLLLLLLLLLLLL
                                              LLLLLLLLLLLLLLLLLLLiiiiiiLLLLLLLLLLLLLLLLLt
                                               LLLLLLLLLLLLLLLLLLL1iitLLLLLLLLLLLLLLLLf
                                                tLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLL:
                                                  LLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLt
                                                    LLLLLLLLLLLLLLLLLLLLLLLLLL.
                                                       ;LLLLLLLLLLLLLLLLL:


                                                                        iLLL                               ,
                                                                        iLLL                            :LLL
                  LLLLLL 1LL; LLLi       LLL   ,LLLLLi        fLLLLLi   iLLL tLLLL.       LLLLLf tLL: LLLLLLLLL     iLLLLL.
                LLLLLLLLLLLL;  LLL.     LLL  LLLLLLLLLLL    LLLLLLLLL.  iLLLLLLLLLLL    LLLLLLLLLLLL: LLLLLLLLL   LLLLLLLLLLL
               LLL       LLL;   LLL    LLL: LLLL     ,LLL  LLL,         iLLL     tLLL  LLL       LLL:   :LLL     LLL:     LLLL
              ,LLL       LLL;   .LLL  LLLt  LLL       LLL  LLL          iLLL      LLL ,LLL       LLL:   :LLL     LLL       LLL
               LLL       LLL;    iLLf,LLL   LLLf      LLL  LLL          iLLL      LLL  LLL       LLL:   :LLL     LLL.     fLLL
               ,LLLLLtfLLLLL;     LLLLLL     LLLLLLLLLLL    LLLLLLLLLt  iLLL      LLL  ,LLLLLtLLLLLL:    LLLLLL   LLLLLLLLLLL
                 ,LLLLLL iLL;      LLLL        iLLLLLf        LLLLLLf   iLLL      LLL    ,LLLLLL 1LL:     iLLLLf    fLLLLL1

`)
	fmt.Printf("%s", reset)
}
