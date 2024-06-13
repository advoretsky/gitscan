package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strings"

	"github.com/google/go-github/v41/github"
	"github.com/jackc/pgx/v4/pgxpool"
	"golang.org/x/oauth2"
)

// Database connection pool
var pool *pgxpool.Pool

// Function to initialize database connection pool
func initDB() error {
	connString := "postgresql://gitscan:123456@localhost:5432/gitscan"
	var err error
	pool, err = pgxpool.Connect(context.Background(), connString)
	if err != nil {
		return err
	}

	return nil
}

func ScanCommits(owner, repo, branchName string) {
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		log.Fatalf("failed to aquire a db connection: %w", err)
	}
	tokenBytes, err := ioutil.ReadFile(".github_token")
	if err != nil {
		log.Fatalf("Error reading token file: %v", err)
	}

	token := strings.TrimSpace(string(tokenBytes))

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	opts := &github.CommitsListOptions{SHA: branchName}

	scanner := NewScanner()

	for {
		commits, resp, err := client.Repositories.ListCommits(ctx, owner, repo, opts)
		if err != nil {
			if _, ok := err.(*github.ErrorResponse); ok {
				log.Fatalf("Error fetching commits: %v. Check if the repository or branch exists.", err)
			} else {
				log.Fatalf("Error fetching commits: %v", err)
			}
		}

		for _, commit := range commits {
			info, err := getCommitInfo(conn, owner, repo, commit.GetSHA())
			if err != nil {
				log.Fatalf("failed ot read from DB: %w", err)
			}
			if info != nil {
				if len(*info) > 0 {
					fmt.Printf("Commit %s was already processed: %s\n", commit.GetSHA(), *info)
				} else {
					fmt.Printf("Commit %s was already processed\n", commit.GetSHA())
				}
				continue
			}
			fmt.Printf("Commit: %s, Author: %s, Date: %s\n", commit.GetSHA(), commit.GetCommit().GetAuthor().GetName(), commit.GetCommit().GetAuthor().GetDate())

			// Fetch commit details to get the files changed
			commitDetails, _, err := client.Repositories.GetCommit(ctx, owner, repo, commit.GetSHA(), nil)
			if err != nil {
				log.Fatalf("Error fetching commit details: %v", err)
			}

			var newInfo string
			// Iterate over the files in the commit and check for secret leaks
			for _, file := range commitDetails.Files {
				// fmt.Printf("File: %s\n", file.GetFilename())
				patch := file.GetPatch()
				lines := strings.Split(patch, "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "+") {
						// Here, you can add your logic to detect secrets in the added lines
						if scanner.ContainsSecret(line) {
							// TODO collect here all found leaks
							newInfo = fmt.Sprintf("Potential secret leak detected in file %s:\n%s\n", file.GetFilename(), line)
							fmt.Println(info)
						}
					}
				}
			}
			if err = saveCommit(conn, owner, repo, commit.GetSHA(), newInfo); err != nil {
				log.Fatalf("failed to update DB: %w", err)
			}
		}
		fmt.Println("batch is processed")

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
}

// Function to save commit information to the database
func saveCommit(conn *pgxpool.Conn, owner, repo, commit, info string) error {
	_, err := conn.Exec(context.Background(), `
		INSERT INTO commits(owner, repo, commit, info)
		VALUES($1, $2, $3, $4)
		ON CONFLICT (owner, repo, commit) DO UPDATE
		SET info = EXCLUDED.info`,
		owner, repo, commit, info)
	return err
}

// Function to retrieve commit info from the database
func getCommitInfo(conn *pgxpool.Conn, owner, repo, commit string) (*string, error) {
	var info string
	err := conn.QueryRow(context.Background(), `
		SELECT info
		FROM commits
		WHERE owner = $1 AND repo = $2 AND commit = $3`,
		owner, repo, commit).Scan(&info)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil // Return nil if no rows found (commit not in DB)
		}
		return nil, err
	}
	return &info, nil
}

type Scanner struct {
	secretRegexp           *regexp.Regexp
	biggerSecretRegexp     *regexp.Regexp
	base64specificSymbols1 *regexp.Regexp
	base64specificSymbols2 *regexp.Regexp
	base64specificSymbols3 *regexp.Regexp
	base64specificSymbols4 *regexp.Regexp
}

func NewScanner() *Scanner {
	return &Scanner{
		secretRegexp:           regexp.MustCompile(`[0-9a-zA-Z/+]{40}`),
		biggerSecretRegexp:     regexp.MustCompile(`[0-9a-zA-Z/+]{42}`),
		base64specificSymbols1: regexp.MustCompile("[/+]"),
		base64specificSymbols2: regexp.MustCompile("[0-9]"),
		base64specificSymbols3: regexp.MustCompile("[a-z]"),
		base64specificSymbols4: regexp.MustCompile("[A-Z]"),
	}
}

func (s Scanner) ContainsSecret(line string) bool {
	matches := s.secretRegexp.FindStringSubmatch(line)
	if len(matches) == 0 {
		return false
	}

	//// TODO improve that not completely correct false positive prevention
	//if s.biggerSecretRegexp.MatchString(line) {
	//	return false
	//}
	//
	//// to decrease a chance of false positive on expense of a small risk to miss a real secret
	//// we expect that random secret will contain at least one small, big letter, digit and one of non symbols
	//for _, m := range matches {
	//	if !s.base64specificSymbols1.MatchString(m) {
	//		return false
	//	}
	//	if !s.base64specificSymbols2.MatchString(m) {
	//		return false
	//	}
	//	if !s.base64specificSymbols3.MatchString(m) {
	//		return false
	//	}
	//}
	return true
}

func main() {
	owner := "Homebrew"    // replace with the actual owner name
	repo := "brew"         // replace with the actual repo name
	branchName := "master" // replace with the actual branch name

	err := initDB()
	if err != nil {
		log.Fatalf("failed to init DB: %w", err)
	}

	ScanCommits(owner, repo, branchName)
}
