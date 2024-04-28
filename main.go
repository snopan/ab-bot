package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/playwright-community/playwright-go"
	tmpmailgo "github.com/snopan/tmpmail-go"
)

type RegisterPayload struct {
	Account     string `json:"account"`
	AccountType int    `json:"account_type"`
}

type SendCodePayload struct {
	Account        string `json:"account"`
	AccountType    int    `json:"account_type"`
	CodeType       int    `json:"code_type"`
	SupportCaptcha int    `json:"support_captcha"`
}

type ResponseStatus struct {
	Msg string `json:"msg"`
}

func main() {
	lambda.Start(HandleRequest)
}

func HandleRequest() (string, error) {
	if err := playwright.Install(); err != nil {
		return "", fmt.Errorf("failed to install playwright deps: %w", err)
	}

	email, err := tmpmailgo.NewEmail()
	if err != nil {
		panic(err)
	}

	fmt.Println("got email: " + email.String())

	fmt.Println("going to page...")
	page, err := gotoPage("https://www.arenabreakoutinfinite.com/en/index.html")
	if err != nil {
		return "", fmt.Errorf("failed to go to page: %w", err)
	}

	fmt.Println("sending code...")
	if err = sendCode(page, email.String()); err != nil {
		return "", fmt.Errorf("failed to send code: %w", err)
	}

	fmt.Println("waiting for verification code...")
	code, err := getVerification(email)
	if err != nil {
		return "", fmt.Errorf("failed to wait for code: %w", err)
	}
	fmt.Println("got verification code: " + code)

	fmt.Println("registering...")
	if err = register(page, code); err != nil {
		return "", fmt.Errorf("failed to register: %w", err)
	}

	fmt.Println("getting extra draws...")
	if err = getExtraDraws(page); err != nil {
		return "", fmt.Errorf("failed to get extra draws: %w", err)
	}

	fmt.Println("drawing rewards...")
	if err = drawReward(page); err != nil {
		return "", fmt.Errorf("failed to draw rewards: %w", err)
	}

	fmt.Println("checking rewards...")
	rewards, err := listRewards(page)
	if err != nil {
		return "", fmt.Errorf("failed to list rewards: %w", err)
	}
	fmt.Println(rewards)

	return rewards, nil
}

func gotoPage(url string) (playwright.Page, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("could not start playwright: %w", err)
	}

	headless := true
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: &headless,
	})
	if err != nil {
		return nil, fmt.Errorf("could not launch browser: %w", err)
	}

	page, err := browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("could not create page: %w", err)
	}

	if _, err = page.Goto(url); err != nil {
		return nil, fmt.Errorf("could not goto: %w", err)
	}

	// This cookies tab blocks the register button
	_ = page.Locator("#onetrust-reject-all-handler").Click()

	return page, nil
}

func sendCode(page playwright.Page, email string) error {
	if err := page.Locator("#beginLogin").Click(); err != nil {
		return fmt.Errorf("could not start register: %w", err)
	}

	if err := page.Locator(".login-goRegister__button").Click(); err != nil {
		return fmt.Errorf("could not go to register: %w", err)
	}

	if err := page.Locator("#registerForm_account").Fill(email); err != nil {
		return fmt.Errorf("could not fill email: %w", err)
	}

	if err := page.Locator("._1egsyt72").Click(); err != nil {
		return fmt.Errorf("could not send code: %w", err)
	}

	return nil
}

func getVerification(email tmpmailgo.Email) (string, error) {
	maxTries := 10
	tk := time.NewTicker(5 * time.Second)
	for range tk.C {

		mail, err := email.GetInbox()
		if err != nil {
			return "", fmt.Errorf("failed to get latest message: %w", err)
		}
		fmt.Printf("refreshing inbox found %d emails\n", len(mail))

		for _, m := range mail {
			fromParts := strings.Split(m.From, "@")
			if len(fromParts) != 2 {
				return "", fmt.Errorf("got invalid email should not happen: %s", m.From)
			}

			if fromParts[1] == "LevelInfinitePass.account.levelinfinite.com" {
				return m.Subject[:5], nil
			}
		}

		maxTries--
		if maxTries == 0 {
			break
		}
	}

	return "", errors.New("timed out")
}

func register(page playwright.Page, code string) error {
	if err := page.Locator("._1egsyt71").Locator(".infinite-input").Fill(code); err != nil {
		return fmt.Errorf("could not fill in code: %w", err)
	}

	if err := page.Locator("._1462jlh1").Locator(".infinite-checkbox-input").Click(); err != nil {
		return fmt.Errorf("could not check confirmation: %w", err)
	}

	if _, err := page.Screenshot(playwright.PageScreenshotOptions{
		Path: playwright.String("foo.png"),
	}); err != nil {
		log.Fatalf("could not create screenshot: %v", err)
	}

	if err := page.Locator("._1aucblfa").Locator(".infinite-btn").Click(); err != nil {
		return fmt.Errorf("could not confirm register: %w", err)
	}

	if err := page.Locator("#dlg_beta_signup").Locator(".dg_box_close").Click(); err != nil {
		return fmt.Errorf("could not close modal after register: %w", err)
	}

	return nil
}

func getExtraDraws(page playwright.Page) error {
	for i := 2; i < 5; i++ {
		if err := page.Locator(fmt.Sprintf("#task_%d", i)).Click(); err != nil {
			return fmt.Errorf("could not click task %d: %w", i, err)
		}
	}

	page.WaitForTimeout(1000)

	if err := page.BringToFront(); err != nil {
		return fmt.Errorf("could not bring page to front: %w", err)
	}

	if err := page.Locator("#task_5").Click(); err != nil {
		return fmt.Errorf("could not click task 5: %w", err)
	}

	if err := page.Locator("#dlg_share_link").Locator(".dg_box_close").Click(); err != nil {
		return fmt.Errorf("could not click out of share link: %w", err)
	}

	return nil
}

func drawReward(page playwright.Page) error {
	if err := page.Locator(".draw_reward").Locator(".reward_list_awaits_btn").Click(); err != nil {
		return fmt.Errorf("could not open draw modal: %w", err)
	}

	for i := 0; i < 5; i++ {
		if err := page.Locator("#draw_now").Click(); err != nil {
			return fmt.Errorf("could draw reward: %w", err)
		}

		if err := page.Locator("#back_to_draw").Click(); err != nil {
			return fmt.Errorf("could go back to draw: %w", err)
		}
	}

	if err := page.Locator(".dlg_draw_now").Locator(".dg_box_close").Click(); err != nil {
		return fmt.Errorf("could not close draw modal: %w", err)
	}

	return nil
}

func listRewards(page playwright.Page) (string, error) {
	if err := page.Locator(".draw_reward").Locator(".awaits_btn draw_awaits_btn").Click(); err != nil {
		return "", fmt.Errorf("could not reward list modal: %w", err)
	}

	text, err := page.Locator("#showRewardList").TextContent()
	if err != nil {
		return "", fmt.Errorf("could not read reward list: %w", err)
	}

	return text, nil
}
