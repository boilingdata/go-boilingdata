package boilingdata

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/pavi6691/go-boilingdata/constants"
)

type AwsCredentials struct {
	AccessKeyId     string
	SecretAccessKey string
	SessionToken    string
	CredentialScope string
}

type Auth struct {
	userName                        string
	password                        string
	authResult                      *cognitoidentityprovider.AuthenticationResultType
	timeWhenLastJwtTokenWasRecieved time.Time
}

func (s *Auth) GetSignedWssHeader(token string) (http.Header, error) {
	creds, err := GetAwsCredentialss(token)
	if err != nil {
		return nil, err
	}
	header, err := getSignedHeaders(creds)
	if err != nil {
		log.Printf("Error getting singned url headers: " + err.Error())
		return nil, err
	}
	return header, err
}

func (s *Auth) GetSignedWssUrl(headers http.Header) (string, error) {
	credential, signature, err := extractCredentialAndSignature(headers["Authorization"][0])
	if err != nil {
		log.Printf("Error Extracting Credential and Signature: " + err.Error())
		return "", err
	}
	signedUrl := constants.WssUrl + "?" + fmt.Sprintf(constants.SignWrlFormat, url.QueryEscape(credential)+"&",
		url.QueryEscape(headers["X-Amz-Date"][0])+"&", url.QueryEscape(headers["X-Amz-Security-Token"][0])+"&", url.QueryEscape(signature))
	return signedUrl, nil
}

func extractCredentialAndSignature(header string) (string, string, error) {
	credentialStart := strings.Index(header, "Credential=")
	if credentialStart == -1 {
		return "", "", fmt.Errorf("credential not found in header")
	}
	credentialStart += len("Credential=")
	credentialEnd := strings.Index(header[credentialStart:], ",") + credentialStart
	if credentialEnd == -1 {
		return "", "", fmt.Errorf("credential end not found in header")
	}

	signatureStart := strings.Index(header, "Signature=")
	if signatureStart == -1 {
		return "", "", fmt.Errorf("signature not found in header")
	}
	signatureStart += len("Signature=")
	signatureEnd := len(header)

	return header[credentialStart:credentialEnd], header[signatureStart:signatureEnd], nil
}

func GetAwsCredentialss(jwtIdToken string) (AwsCredentials, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(constants.Region))
	if err != nil {
		return AwsCredentials{}, fmt.Errorf("failed to load configuration, %v", err)
	}
	cognitoClient := cognitoidentity.NewFromConfig(cfg)

	out, err := cognitoClient.GetId(context.TODO(), &cognitoidentity.GetIdInput{
		IdentityPoolId: aws.String(constants.IdentityPoolId),
		Logins:         map[string]string{constants.CognitoIdp: jwtIdToken},
	})

	if err != nil {
		log.Printf("Error : " + err.Error())
		return AwsCredentials{}, err
	}

	ctx := context.Background()
	credRes, err := cognitoClient.GetCredentialsForIdentity(ctx, &cognitoidentity.GetCredentialsForIdentityInput{
		IdentityId: out.IdentityId,
		Logins: map[string]string{
			constants.CognitoIdp: jwtIdToken,
		},
	})

	if err != nil {
		log.Printf("Error : " + err.Error())
		return AwsCredentials{}, err
	}

	awsCreds := AwsCredentials{
		AccessKeyId:     *credRes.Credentials.AccessKeyId,
		SecretAccessKey: *credRes.Credentials.SecretKey,
		SessionToken:    *credRes.Credentials.SessionToken,
	}

	return awsCreds, nil
}

func (auth *Auth) GetAWSSingingHeaders(urlString string) (http.Header, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}

	// Extract query parameters
	query := u.Query()

	// Extract date
	amzDate := query.Get("X-Amz-Date")

	// Extract signed headers

	// signedHeaders := query.Get("X-Amz-SignedHeaders")

	// Extract algorithm
	algorithm := query.Get("X-Amz-Algorithm")

	// Extract credential
	credential := query.Get("X-Amz-Credential")

	// Extract security token
	securityToken := query.Get("X-Amz-Security-Token")

	// Extract signature
	signature := query.Get("X-Amz-Signature")

	// Prepare headers
	headers := map[string][]string{
		"Authorization": {
			fmt.Sprintf("%s Credential=%s, SignedHeaders=%s, Signature=%s", algorithm, credential, "host;x-amz-date;x-amz-security-token,", signature),
		},
		"X-Amz-Date":           {amzDate},
		"X-Amz-Security-Token": {securityToken},
	}

	return headers, nil
}

func getSignedHeaders(creds AwsCredentials) (http.Header, error) {
	// Create a signer with the given AWS credentials
	signer := v4.NewSigner(credentials.NewStaticCredentials(creds.AccessKeyId, creds.SecretAccessKey, creds.SessionToken))
	wsURL := constants.WssUrl
	req, err := http.NewRequest("GET", wsURL, nil)
	if err != nil {
		return nil, err
	}
	// Sign the request
	_, err = signer.Sign(req, nil, constants.Service, constants.Region, time.Now())
	if err != nil {
		log.Println("Error signing request:", err)
		return nil, nil
	}
	// Return the signed URL
	return req.Header, err
}

func (auth *Auth) Authenticate() (string, error) {
	muLock.Lock()
	defer muLock.Unlock()

	var authInput *cognitoidentityprovider.InitiateAuthInput
	if auth.IsUserLoggedIn() && !auth.IsTokenExpired() {
		return *auth.authResult.IdToken, nil
	} else if auth.IsUserLoggedIn() || auth.password == "" {
		log.Println("Token expired, Getting token with refresh token..")
		authInput = &cognitoidentityprovider.InitiateAuthInput{
			AuthFlow: aws.String("REFRESH_TOKEN_AUTH"),
			AuthParameters: map[string]*string{
				"REFRESH_TOKEN": aws.String(*auth.authResult.RefreshToken),
				"POOL_ID":       aws.String(constants.PoolID),
			},
			ClientId: aws.String(constants.ClientID),
		}
	} else {
		log.Println("Logging in..")
		// Authenticate user
		authInput = &cognitoidentityprovider.InitiateAuthInput{
			AuthFlow: aws.String("USER_PASSWORD_AUTH"),
			AuthParameters: map[string]*string{
				"USERNAME": aws.String(auth.userName),
				"PASSWORD": aws.String(auth.password),
				"POOL_ID":  aws.String(constants.PoolID),
			},
			ClientId: aws.String(constants.ClientID),
		}
	}
	//
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(constants.Region)},
	)
	if err != nil {
		return "", err
	}
	cognitoClient := cognitoidentityprovider.New(sess)
	authOutput, err := cognitoClient.InitiateAuth(authInput)
	//
	if err != nil {
		log.Println("Login unsucessful, ->" + err.Error())
		auth.authResult = nil
		RemoveUser(auth.userName)
		return "", err
	}
	auth.timeWhenLastJwtTokenWasRecieved = time.Now()
	auth.authResult = authOutput.AuthenticationResult
	// Handle MFA challenges if required
	if authOutput.ChallengeName != nil {
		switch *authOutput.ChallengeName {
		case "SMS_MFA":
			mfaCode, err := promptMFA("Please enter MFA (sms)")
			if err != nil {
				RemoveUser(auth.userName)
				return "", err
			}
			err = sendMFA(cognitoClient, authOutput.Session, mfaCode, "SMS_MFA")
			if err != nil {
				RemoveUser(auth.userName)
				return "", err
			}
		case "SOFTWARE_TOKEN_MFA":
			mfaCode, err := promptMFA("Please enter MFA (totp)")
			if err != nil {
				RemoveUser(auth.userName)
				return "", err
			}
			err = sendMFA(cognitoClient, authOutput.Session, mfaCode, "SOFTWARE_TOKEN_MFA")
			if err != nil {
				RemoveUser(auth.userName)
				return "", err
			}
		}
	}

	// Handle newPasswordRequired if required
	if authOutput.AuthenticationResult == nil {
		newPassword, err := promptPassword("Please enter new password")
		if err != nil {
			RemoveUser(auth.userName)
			return "", err
		}
		// Assume, need to provide new password
		// we might need additional logic to determine if new password is required
		return completeNewPasswordChallenge(cognitoClient, authOutput.Session, newPassword)
	}
	// Authentication successful
	log.Println("Authentication successful")
	return *authOutput.AuthenticationResult.IdToken, nil
}

func (auth *Auth) IsUserLoggedIn() bool {
	if auth.authResult != nil && auth.authResult.IdToken != nil {
		return true
	}
	return false
}
func (auth *Auth) IsTokenExpired() bool {
	if auth.authResult != nil && auth.authResult.ExpiresIn != nil {
		expirationTime := auth.timeWhenLastJwtTokenWasRecieved.Add(time.Second * time.Duration(*auth.authResult.ExpiresIn))
		if time.Now().Unix() < expirationTime.Unix() {
			return false
		}
	}
	return true
}

func promptMFA(promptMsg string) (string, error) {
	// Implement logic for prompting MFA from the user
	return "", errors.New("Prompting for MFA not implemented")
}

func sendMFA(client *cognitoidentityprovider.CognitoIdentityProvider, session *string, mfaCode, mfaType string) error {
	input := &cognitoidentityprovider.RespondToAuthChallengeInput{
		ChallengeName: aws.String("SMS_MFA"), // or "SOFTWARE_TOKEN_MFA"
		ClientId:      aws.String("YOUR_CLIENT_ID"),
		Session:       session,
		ChallengeResponses: map[string]*string{
			"SMS_MFA_CODE": aws.String(mfaCode), // or "SOFTWARE_TOKEN_MFA_CODE"
		},
	}
	_, err := client.RespondToAuthChallenge(input)
	return err
}

func promptPassword(promptMsg string) (string, error) {
	// Implement logic for prompting new password from the user
	return "", errors.New("Prompting for new password not implemented")
}

func completeNewPasswordChallenge(client *cognitoidentityprovider.CognitoIdentityProvider, session *string, newPassword string) (string, error) {
	// Assume that you need to complete new password challenge
	// we might need additional logic here
	return "", nil
}
