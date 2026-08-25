# Contributing to Log Shuttle

## Governance Model

Log Shuttle is published but not supported. It is shared because its code and
concepts may be useful to the open source community, but the maintainers do not
actively solicit contributions or guarantee review, support, or release
timelines.

## Issues and Pull Requests

Use [GitHub Issues](https://github.com/heroku/log-shuttle/issues) to report
bugs or discuss proposed changes. If you choose to contribute, open a pull
request against the `master` branch of
[heroku/log-shuttle](https://github.com/heroku/log-shuttle).

Please keep pull requests focused, describe the problem and approach, and
include tests for behavior changes. Maintainers may close or leave pull
requests unanswered when they do not fit the project's limited maintenance
model.

## Testing

Run the repository's Go test suite before submitting a pull request:

```bash
go test -v ./...
go test -v -race ./...
```

## Contributor License Agreement

Contributions require a signed [Salesforce Contributor License Agreement](https://cla.salesforce.com/sign-cla).
You only need to sign the CLA once for Salesforce open source projects.

## Code of Conduct

All participants must follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## License

By contributing, you agree that your contribution is licensed under the
project license in the [License section of readme.md](readme.md#license).
