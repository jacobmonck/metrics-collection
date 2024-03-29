package calico

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/jacobmonck/metrics-collection/src/api/db"
	"github.com/sirupsen/logrus"
	"time"
)

func (b *Bot) SyncGuild(guild discord.Guild) {
	b.GuildSync.Synced = false

	logrus.Info("Synchronizing guild members...")

	apiStart := time.Now()
	members, err := b.Client.MemberChunkingManager().RequestMembersWithQuery(
		guild.ID,
		"",
		guild.MemberCount,
	)
	if err != nil {
		logrus.WithError(err).Error("Failed to fetch members from the Discord API.")
	}
	apiDuration := time.Since(apiStart)

	logrus.Infof(
		"Fetched %d members from the Discord API in %s.",
		len(members),
		apiDuration,
	)

	logrus.Info("Updating members in database...")

	dbStart := time.Now()
	db.BulkUpsertMembers(members)
	dbDuration := time.Since(dbStart)

	logrus.Infof("Synchronized members with the database in %s.", dbDuration)

	b.SyncChannels(guild)
}

func (b *Bot) SyncChannels(guild discord.Guild) {
	b.GuildSync.Synced = false

	logrus.Info("Synchronizing guild channels...")

	rest := b.Client.Rest()

	channels, err := rest.GetGuildChannels(guild.ID)
	if err != nil {
		logrus.Fatalf("Failed to synchronize channels: %s", err)
	}

	var categoryChannels []discord.GuildCategoryChannel
	var textChannels []discord.GuildTextChannel
	var threadChannels []discord.GuildThread

	for _, channel := range channels {
		switch channel.Type() {
		case discord.ChannelTypeGuildCategory:
			categoryChannels = append(
				categoryChannels,
				channel.(discord.GuildCategoryChannel),
			)
		case discord.ChannelTypeGuildText:
			textChannels = append(textChannels, channel.(discord.GuildTextChannel))
		case discord.ChannelTypeGuildPublicThread:
			threadChannels = append(threadChannels, channel.(discord.GuildThread))
		}
	}

	db.UpdateChannels(categoryChannels, textChannels, threadChannels)

	logrus.Info("Finished synchronizing guild channels.")

	b.GuildSync.Synced = true
}
