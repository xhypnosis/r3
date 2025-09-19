package mail_receive

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"r3/cache"
	"r3/config"
	"r3/db"
	"r3/log"
	"r3/tools"
	"r3/types"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
)

var (
	accountMode   = "imap"
	collectPerRun = uint32(50)
	imapFolder    = "INBOX"
	regexCid      = regexp.MustCompile(`<img[^>]*cid\:([^\"]*)`)
)

func DoAll() error {
	if !cache.GetMailAccountsExist() {
		log.Info(log.ContextMail, "cannot start retrieval, no accounts defined")
		return nil
	}

	accountMap := cache.GetMailAccountMap()

	for _, ma := range accountMap {
		if ma.Mode != accountMode {
			continue
		}

		log.Info(log.ContextMail, fmt.Sprintf("is retrieving from '%s'", ma.Name))

		if err := do(ma); err != nil {
			log.Error(log.ContextMail, fmt.Sprintf("failed to retrieve from '%s'", ma.Name), err)
			continue
		}
	}
	return nil
}

func do(ma types.MailAccount) error {

	// get OAuth client token if used
	usesXoauth2 := ma.OauthClientId.Valid
	if usesXoauth2 {
		if !config.GetLicenseActive() {
			return errors.New("no valid license (required for OAuth clients)")
		}
		c, err := cache.GetOauthClient(ma.OauthClientId.Int32)
		if err != nil {
			return err
		}
		if !c.ClientSecret.Valid || !c.TokenUrl.Valid {
			return errors.New("missing client secret or token URL in OAUTH client")
		}
		ma.Password, err = tools.GetOAuthToken(c.ClientId, c.ClientSecret.String, c.TokenUrl.String, c.Scopes)
		if err != nil {
			return err
		}
	}

	// start IMAP client
	var c *client.Client
	var err error

	// STARTTLS starts with unencrypted connection then upgrades
	// non-STARTTLS starts with encrypted connection
	if ma.StartTls {
		c, err = client.Dial(fmt.Sprintf("%s:%d", ma.HostName, ma.HostPort))
	} else {
		c, err = client.DialTLS(fmt.Sprintf("%s:%d", ma.HostName, ma.HostPort), nil)
	}
	if err != nil {
		return err
	}
	defer c.Logout()

	// STARTTLS upgrade to encrypted connection
	if ma.StartTls {
		if err := c.StartTLS(&tls.Config{ServerName: ma.HostName}); err != nil {
			return err
		}
	}

	if usesXoauth2 {
		if err := c.Authenticate(newXoauth2Client(ma.Username, ma.Password)); err != nil {
			return err
		}
	} else {
		if err := c.Login(ma.Username, ma.Password); err != nil {
			return err
		}
	}

	mbox, err := c.Select(imapFolder, false)
	if err != nil {
		return err
	}

	log.Info(log.ContextMail, fmt.Sprintf("found %d messages inside %s for account '%s'",
		mbox.Messages, imapFolder, ma.Name))

	if mbox.Messages == 0 {
		return nil
	}

	log.Info(log.ContextMail, fmt.Sprintf("is now fetching messages (at most %d per run)", collectPerRun))

	// fetch mails from mailbox
	seqDel := new(imap.SeqSet) // messages to delete
	seqGet := new(imap.SeqSet) // messages to fetch

	// fetch X last messages
	offsetFrom := uint32(1)
	offsetTo := mbox.Messages
	if mbox.Messages > collectPerRun-1 {
		offsetFrom = mbox.Messages - (collectPerRun - 1)
	}
	seqGet.AddRange(offsetFrom, offsetTo)

	section := imap.BodySectionName{}
	messages := make(chan *imap.Message, 10)
	doneErr := make(chan error, 1)

	go func() {
		doneErr <- c.Fetch(seqGet, []imap.FetchItem{section.FetchItem()}, messages)
	}()

	// process and then store messages to mail spooler
	for msg := range messages {
		if err := processMessage(ma.Id, msg, &section); err != nil {
			// mail processing can fail because of many reasons, warn and move on
			log.Warning(log.ContextMail, "failed to process message - its not being deleted from the mailbox", err)

		} else {
			// add to deletion sequence if processed successfully
			seqDel.AddNum(msg.SeqNum)
		}
	}

	// wait for fetch to complete
	if err := <-doneErr; err != nil {
		return err
	}

	log.Info(log.ContextMail, fmt.Sprintf("processed %d messages successfully, marking them for deletion",
		len(seqDel.Set)))

	// if database update was successful, execute mail deletion
	if len(seqDel.Set) != 0 {
		item := imap.FormatFlagsOp(imap.AddFlags, true)
		flags := []interface{}{imap.DeletedFlag}
		if err := c.Store(seqDel, item, flags, nil); err != nil {
			return err
		}
		if err := c.Expunge(nil); err != nil {
			return err
		}
	}
	return nil
}

func processMessage(mailAccountId int32, msg *imap.Message, section *imap.BodySectionName) error {

	if msg == nil {
		return errors.New("server did not return message")
	}

	msgBody := msg.GetBody(section)
	if msgBody == nil {
		return errors.New("message body was empty")
	}

	mr, err := mail.CreateReader(msgBody)
	if err != nil {
		return err
	}

	// parse header
	var date time.Time
	var subject string
	var cc, from, to []*mail.Address

	header := mr.Header
	date, err = header.Date()
	if err != nil {
		return err
	}
	from, err = header.AddressList("From")
	if err != nil {
		return err
	}
	to, err = header.AddressList("To")
	if err != nil {
		return err
	}
	cc, err = header.AddressList("Cc")
	if err != nil {
		return err
	}
	subject, err = header.Subject()
	if err != nil {
		return err
	}

	// parse body
	type cid struct {
		contentId   string
		contentType string
		file        []byte
	}
	var body string
	var cids []cid
	var files []types.MailFile
	var gotHtmlText bool = false

	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:

			headerType, _, err := h.ContentType()
			if err != nil {
				return err
			}

			if strings.Contains(headerType, "text") {

				// some senders include both HTML and plain text - in these cases, we only want the HTML version
				if gotHtmlText {
					continue
				}

				b, err := io.ReadAll(p.Body)
				if err != nil {
					return err
				}

				if headerType == "text/plain" {
					// replace 2 new lines with a paragraph, 1 new line with a line break
					body = regexp.MustCompile(`(.*)(\r\n){2,}`).ReplaceAllString(string(b), "<p>$1</p>")
					body = regexp.MustCompile(`(.*)(\n){2,}`).ReplaceAllString(body, "<p>$1</p>")
					body = regexp.MustCompile(`[\r\n]+`).ReplaceAllString(body, "<br />")
				} else {
					body = string(b)
				}

				if headerType == "text/html" {
					gotHtmlText = true
				}

			} else if strings.Contains(headerType, "image") {
				b, err := io.ReadAll(p.Body)
				if err != nil {
					return err
				}

				base64Text := make([]byte, base64.StdEncoding.EncodedLen(len(b)))
				base64.StdEncoding.Encode(base64Text, b)

				id := strings.TrimPrefix(p.Header.Get("Content-ID"), "<")
				id = strings.TrimSuffix(id, ">")

				cids = append(cids, cid{
					contentId:   id,
					contentType: headerType,
					file:        base64Text,
				})
			}

		case *mail.AttachmentHeader:
			name, err := h.Filename()
			if err != nil {
				return err
			}

			b, err := io.ReadAll(p.Body)
			if err != nil {
				return err
			}

			if name == "" {
				// name is not always given, regular case: Outlook forwards messages without file name
				contentType, _, err := h.ContentType()
				if err == nil && contentType == "message/rfc822" {
					name = "ForwardedMessage.eml"
				}
			}

			files = append(files, types.MailFile{
				File: b,
				Name: name,
				Size: int64(len(b) / 1024),
			})
		}
	}
	// look for CID links in text body
	// example: <img src="cid:part1.rDMcVMnB.49OoxErI@rei3.de" alt="">
	for _, matches := range regexCid.FindAllStringSubmatch(body, -1) {

		// match 0: img tag until CID end (<img src="cid:part1.rDMcVMnB.49OoxErI@rei3.de)
		// match 1: CID                   (part1.rDMcVMnB.49OoxErI@rei3.de)
		if len(matches) != 2 {
			continue
		}

		for _, cid := range cids {
			if cid.contentId == matches[1] {
				body = strings.Replace(body, fmt.Sprintf("cid:%s", matches[1]),
					fmt.Sprintf(`data:%s;base64,%s`, cid.contentType, cid.file), -1)

				break
			}
		}
	}

	ctx, ctxCanc := context.WithTimeout(context.Background(), db.CtxDefTimeoutSysTask)
	defer ctxCanc()

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// log to mail traffic log
	fileList := make([]string, 0)
	for _, file := range files {
		fileList = append(fileList, fmt.Sprintf("%s (%dkb)", file.Name, file.Size))
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO instance.mail_traffic (from_list, to_list, cc_list,
			subject, date, files, mail_account_id, outgoing)
		VALUES ($1,$2,$3,$4,$5,$6,$7,FALSE)
	`, getStringListFromAddress(from), getStringListFromAddress(to), getStringListFromAddress(cc),
		subject, date.Unix(), fileList, mailAccountId); err != nil {

		return fmt.Errorf("%w, %s", errors.New("failed to store message in traffic log"), err)
	}

	// store message in spooler
	var mailId int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO instance.mail_spool (from_list, to_list, cc_list,
			subject, body, date, mail_account_id, outgoing)
		VALUES ($1,$2,$3,$4,$5,$6,$7,FALSE)
		RETURNING id
	`, getStringListFromAddress(from), getStringListFromAddress(to), getStringListFromAddress(cc),
		subject, body, date.Unix(), mailAccountId).Scan(&mailId); err != nil {

		return fmt.Errorf("%w, %s", errors.New("failed to store message in spooler"), err)
	}

	// add attachments to spooler
	for i, file := range files {
		if _, err := tx.Exec(ctx, `
			INSERT INTO instance.mail_spool_file (mail_id, position, file, file_name, file_size)
			VALUES ($1,$2,$3,$4,$5)
		`, mailId, i, file.File, file.Name, file.Size); err != nil {
			return fmt.Errorf("%w, %s", errors.New("failed to store message attachment in spooler"), err)
		}
	}
	return tx.Commit(ctx)
}

// helpers
func getStringListFromAddress(list []*mail.Address) string {
	out := make([]string, 0)
	for _, a := range list {
		if a.String() == "" {
			continue
		}
		out = append(out, a.String())
	}
	return strings.Join(out, ", ")
}
