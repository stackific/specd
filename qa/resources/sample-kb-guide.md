# JWT Best Practices and Implementation Guide

JSON Web Tokens (JWT) are a compact, URL-safe means of representing claims to be transferred between two parties. This guide covers the best practices for implementing JWT-based authentication in modern applications.

## Token Structure

A JWT consists of three parts separated by dots:

1. **Header**: Contains the token type and signing algorithm
2. **Payload**: Contains the claims (statements about an entity)
3. **Signature**: Used to verify the token hasn't been altered

```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.
eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.
SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
```

## Signing Algorithms

Always verify the signature before trusting any claims. Libraries that skip signature verification when the algorithm header is set to "none" are vulnerable to serious exploits.

### Symmetric Algorithms (HMAC)

- **HS256**: HMAC using SHA-256. Suitable for most applications.
- **HS384**: HMAC using SHA-384. Higher security margin.
- **HS512**: HMAC using SHA-512. Maximum security for symmetric signing.

### Asymmetric Algorithms (RSA, ECDSA)

- **RS256**: RSA signature with SHA-256. Widely supported.
- **ES256**: ECDSA using P-256 curve and SHA-256. Smaller tokens.
- **EdDSA**: Edwards-curve DSA. Modern and efficient.

## Claim Best Practices

### Registered Claims

Always include these standard claims:

- `iss` (Issuer): Identifies who issued the token
- `sub` (Subject): Identifies the principal subject
- `aud` (Audience): Identifies the intended recipients
- `exp` (Expiration): Token expiration time
- `iat` (Issued At): When the token was issued
- `jti` (JWT ID): Unique identifier for the token

### Token Lifetime

Keep access token lifetimes short (5-15 minutes). Use refresh tokens for longer sessions. Never store sensitive data in the token payload as it can be decoded by anyone.

## Storage Recommendations

- **Web applications**: Use HttpOnly, Secure, SameSite cookies
- **Mobile applications**: Use secure storage (Keychain on iOS, Keystore on Android)
- **Never**: Store tokens in localStorage or sessionStorage for sensitive applications

## Common Vulnerabilities

1. **Algorithm confusion**: Attacker changes the algorithm from RS256 to HS256
2. **Missing expiration**: Tokens that never expire become permanent credentials
3. **Insufficient validation**: Not checking all required claims
4. **Key management**: Using weak or predictable signing keys

## Revocation Strategies

Since JWTs are stateless, revocation requires additional mechanisms:

- **Token blacklist**: Maintain a list of revoked token IDs
- **Short-lived tokens**: Reduce the window of vulnerability
- **Token versioning**: Increment a version counter to invalidate all tokens
