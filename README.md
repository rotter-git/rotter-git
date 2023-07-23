# Rotter Git Server

Provide git over SSH, and only git over SSH. Certainly for now.

## Rationale

I want to only ever support commits over SSH on rotter; HTTPS and the Git protocol have their uses, certainly for cloning without being signed up/ logged in, but realistically I don't care about that use case for now.

I want git operations, including read, to be fully auth'd.

Downloading tagged releases may be done via the UI.

### Wont this hurt people using open source go modules?

Yeah, probs- but this is my internal git server for now.
