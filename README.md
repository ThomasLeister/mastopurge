# mastopurge
Purges Mastodon accounts. Removes posts. Makes things clean again.

PLEASE NOTE: MastoPurge WILL NOT WORK IF TWO FACTOR AUTHENTICATION IS ENABLED FOR YOUR MASTODON ACCOUNT!

## Run Linux x64 binary:

    wget https://github.com/ThomasLeister/mastopurge/releases/download/0.0.1/mastopurge
    chmod u+x mastopurge
    ./mastopurge


## Run from source:

(Golang must be set up)

    (change to your Golang source dir)
    git clone https://github.com/ThomasLeister/mastopurge.git
    cd mastopurge
    go get github.com/mattn/go-mastodon
    go run mastopurge.go
