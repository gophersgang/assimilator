package api

import (
	"net/http"

	"github.com/diyan/assimilator/db"
	"github.com/diyan/assimilator/models"
	"github.com/pkg/errors"

	"github.com/labstack/echo"
)

// User ..
type User struct {
	models.User
	AvatarURL string      `json:"avatarUrl"`
	Options   UserOptions `json:"options"`
}

// UserOptions ..
type UserOptions struct {
	Timezone        string `json:"timezone"`        // TODO double check this
	StacktraceOrder string `json:"stacktraceOrder"` // default
	Language        string `json:"language"`
	Clock24Hours    bool   `json:"clock24Hours"`
}

func ProjectMemberIndexGetEndpoint(c echo.Context) error {
	project := GetProject(c)
	db, err := db.FromE(c)
	if err != nil {
		return err
	}
	// TODO not clear what this expr means -> Q(user__is_active=True) | Q(user__isnull=True)
	users := []*User{}
	_, err = db.SelectBySql(`
		select u.*
			from auth_user u
				join sentry_organizationmember om on u.id = om.user_id
				join sentry_organization o on om.organization_id = o.id
		where o.id = ? and u.is_active = true`,
		project.OrganizationID).
		LoadStructs(&users)
	if err != nil {
		return errors.Wrap(err, "can not read project members")
	}
	for _, user := range users {
		user.PostGet()
		// TODO add real implementation
		user.Options.Language = "en"
		user.Options.Timezone = "UTC"
		user.Options.StacktraceOrder = "default"
	}
	// TODO fill user.AvatarURL, user.Options. Check UserSerializer(Serializer) impl
	return c.JSON(http.StatusOK, users)
}

/*

@register(User)
class UserSerializer(Serializer):
    def _get_identities(self, item_list, user):
        if not (env.request and env.request.is_superuser()):
            item_list = [x for x in item_list if x == user]

        queryset = AuthIdentity.objects.filter(
            user__in=item_list,
        ).select_related('auth_provider', 'auth_provider__organization')

        results = {i.id: [] for i in item_list}
        for item in queryset:
            results[item.user_id].append(item)
        return results

    def get_attrs(self, item_list, user):
        avatars = {
            a.user_id: a
            for a in UserAvatar.objects.filter(
                user__in=item_list
            )
        }
        identities = self._get_identities(item_list, user)

        authenticators = Authenticator.objects.bulk_users_have_2fa([i.id for i in item_list])

        data = {}
        for item in item_list:
            data[item] = {
                'avatar': avatars.get(item.id),
                'identities': identities.get(item.id),
                'has2fa': authenticators[item.id],
            }
        return data

    def serialize(self, obj, attrs, user):
        d = {
            'id': six.text_type(obj.id),
            'name': obj.get_display_name(),
            'username': obj.username,
            'email': obj.email,
            'avatarUrl': get_gravatar_url(obj.email, size=32),
            'isActive': obj.is_active,
            'isManaged': obj.is_managed,
            'dateJoined': obj.date_joined,
            'lastLogin': obj.last_login,
            'has2fa': attrs['has2fa'],
        }

        if obj == user:
            options = {
                o.key: o.value
                for o in UserOption.objects.filter(
                    user=user,
                    project__isnull=True,
                )
            }
            stacktrace_order = int(options.get('stacktrace_order', -1) or -1)
            if stacktrace_order == -1:
                stacktrace_order = 'default'
            elif stacktrace_order == 2:
                stacktrace_order = 'newestFirst'
            elif stacktrace_order == 1:
                stacktrace_order = 'newestLast'

            d['options'] = {
                'language': options.get('language') or 'en',
                'stacktraceOrder': stacktrace_order,
                'timezone': options.get('timezone') or settings.SENTRY_DEFAULT_TIME_ZONE,
                'clock24Hours': options.get('clock_24_hours') or False,
            }

        if attrs.get('avatar'):
            avatar = {
                'avatarType': attrs['avatar'].get_avatar_type_display(),
                'avatarUuid': attrs['avatar'].ident if attrs['avatar'].file else None
            }
        else:
            avatar = {'avatarType': 'letter_avatar', 'avatarUuid': None}
        d['avatar'] = avatar

        if attrs['identities'] is not None:
            d['identities'] = [{
                'id': six.text_type(i.id),
                'name': i.ident,
                'organization': {
                    'slug': i.auth_provider.organization.slug,
                    'name': i.auth_provider.organization.name,
                },
                'provider': {
                    'id': i.auth_provider.provider,
                    'name': i.auth_provider.get_provider().name,
                },
                'dateSynced': i.last_synced,
                'dateVerified': i.last_verified,
            } for i in attrs['identities']]

        return d
*/
