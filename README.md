# MastoPurge
*Purges Mastodon accounts. Deletes old posts. Makes things clean again.*

MastoPurge connects to your Mastodon account and automatically mass-deletes your old toots. You define what "old" means: Posts from the last few hours/days/weeks/months are preserved while older ones get deleted from your account.

MastoPurge is executed as a command line application on your own PC. You do not need to rely on third parties.

**Please note:**
* Deleting hundreds or thousands of posts can take a long time due to Mastodon API limits/throttling
* There is no guarantee that your federated toots are deleted on every foreign instance

## Demo Video

See https://youtu.be/fQzc6CHq3aU

## Why should you use this tool?

There is a German word for the process of removing old data: "Datenhygiene". Datenhygiene can be  translated to "data hygiene", which means to remove data which is not needed or relevant anymore. This brings some advantages:

* **Remove part of your personal history from the internet**: Maybe you regret having written something publicly or privately, which new users should not see anymore. We all change our opinions over time. Be sure nobody gets a wrong impression based on outdated posts.  
* **Improve server performance**: Less posts => Less data => Better database performance => Quicker Mastodon reaction. Posts usually are not relevant anymore after a few days. Do your instance administrator a favor and clean up your space to keep costs for computing and storage as low as possible.

## Why should you NOT use this tool?

Mass-deletions by MastoPurge cause a lot of traffic between Mastodon instances, because deletions are federated one after another. Unfortunately Mastodon does not offer mass-deleting old posts itself, so there is no other efficient way to get rid of your old data. Mass-deletions could be implemented quite traffic-respecting, if well integrated into Mastodon - obviously there is no solution to that yet. (Also see: [#875](https://github.com/tootsuite/mastodon/issues/875), [#69](https://github.com/glitch-soc/mastodon/issues/69))

## Download and run Linux x64 binary:

Download latest binary from https://github.com/ThomasLeister/mastopurge/releases/latest

    chmod u+x mastopurge_linux_x86_64
    ./mastopurge_linux_x86_64


## Compile and run from source:

(Golang must be set up)

    (change to your Golang source dir)
    git clone https://github.com/ThomasLeister/mastopurge.git
    cd mastopurge
    go run mastopurge.go


## Usage instructions

1. Download and run MastoPurge (see above)
2. Enter the domain name of your Mastodon home instance
3. MastoPurge will ask you to visit a certain URL. Open this URL in your web browser
4. Authorize MastoPurge to access your Mastodon account. A Code will be displayed.
5. Enter the code into MastoPurge
6. Select a timespan of your choice. Posts from this time range will *not* be deleted. Older posts will be removed. _(Note: "pinned posts" will **not** be deleted!)_
7. Wait. Removing hundreds or thousands of posts can take a long time due to API limits.
8. MastoPurge will remember your account the next time you use it. No more authentication needed. If you want to use another account, delete the `.mastopurgesettings` file.


### Non-interactive mode

After you have run Mastopurge in interactive mode, once (see instructions above), you will be able to run it in non-interactive mode, if you like. This mode enables you to run Mastopurge automatically e.g. as a Crob Job.

Example:

```
./mastopurge --noninteractive --maxage "30 days"
```

### Dry-run mode

If you'd like to check whether Mastopurge works properly (without actually deleting your posts!), you can try the "dry run mode":

```
./mastopurge --noninteractive --maxage "30 days" --dryrun
```

### Favourites

If you want to undo favourites, add the `--favs` parameter:

```
./mastopurge --maxage "365 days" --favs
```

### Verbose output

If you are really curious about some internals, add verbose output:

```
./mastopurge --maxage "365 days" --verbose
````
