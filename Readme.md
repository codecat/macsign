# MacSign
Faster way to do signing and notarization of applications and installers on Mac. Mostly used for [Nimble Tools](https://nimble.tools/) products.

## Usage
Install using:
```
$ go get github.com/codecat/macsign
```

You can pre-configure MacSign by creating `~/.macsign.toml`:
```toml
[keychain]
profile = "Your profile name"

[keychain.identity]
application = "Developer ID Application: Your Name (ABCDCEFGHI)"
installer = "Developer ID Installer: Your Name (ABCDEFGHI)"
```
The configuration file may optionally exist in the current working directory.

Run `macsign` with some paths to sign and notarize with your pre-configured identity. You can provide multiple paths, which are codesigned individually, but notarized together:
```
$ macsign Test.component Test.vst3
```

You can currently sign 2 types of paths:
* Installers as `.pkg` which uses the `keychain.identity.application`
* Anything else such as `.app`, `.component`, `.vst3`, etc.

## Configuring keychain profile
If you don't have a keychain profile configured yet, run this:
```
$ xcrun notarytool store-credentials
```
When asked for an App Store Connect API key, [create one here](https://appstoreconnect.apple.com/access/integrations/api).
