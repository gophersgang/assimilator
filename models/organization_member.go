package models

import "time"

type OrganizationMemberTeam struct {
	ID                   int  `db:"id" json:"id,string"`
	OrganizationMemberID int  `db:"organizationmember_id" json:"organizationmemberId"`
	TeamID               int  `db:"team_id" json:"teamId"`
	IsActive             bool `db:"is_active" json:"isActive"`
}

// OrganizationMember identifies relationships between teams and users.
// Users listed as team members are considered to have access to all projects
// and could be thought of as team owners (though their access level may not)
// be set to ownership.
type OrganizationMember struct {
	ID              int       `db:"id" json:"id,string"`
	OrganizationID  int       `db:"organization_id" json:"organizationId"`
	UserID          int       `db:"user_id" json:"userId"`
	Type            int       `db:"type" json:"type"`
	DateCreated     time.Time `db:"date_added" json:"dateCreated"`
	Email           string    `db:"email" json:"email"`
	HasGlobalAccess bool      `db:"has_global_access" json:"hasGlobalAccess"`
	Flags           int64     `db:"flags" json:"flags"`
	Counter         int       `db:"counter" json:"counter"`
	Role            string    `db:"role" json:"role"`
}

/*
describe sentry_organizationmember_teams
+-----------------------+---------+-------------------------------------------------------------------------------+
| Column                | Type    | Modifiers                                                                     |
|-----------------------+---------+-------------------------------------------------------------------------------|
| id                    | integer |  not null default nextval('sentry_organizationmember_teams_id_seq'::regclass) |
| organizationmember_id | integer |  not null                                                                     |
| team_id               | integer |  not null                                                                     |
| is_active             | boolean |  not null                                                                     |
+-----------------------+---------+-------------------------------------------------------------------------------+


class OrganizationMemberTeam(BaseModel):
    __core__ = True

    id = BoundedAutoField(primary_key=True)
    team = FlexibleForeignKey('sentry.Team')
    organizationmember = FlexibleForeignKey('sentry.OrganizationMember')
    # an inactive membership simply removes the team from the default list
    # but still allows them to re-join without request
    is_active = models.BooleanField(default=True)

    class Meta:
        app_label = 'sentry'
        db_table = 'sentry_organizationmember_teams'
        unique_together = (('team', 'organizationmember'),)

    __repr__ = sane_repr('team_id', 'organizationmember_id')

    def get_audit_log_data(self):
        return {
            'team_slug': self.team.slug,
            'member_id': self.organizationmember_id,
            'email': self.organizationmember.get_email(),
            'is_active': self.is_active,
        }



describe sentry_organizationmember
+-------------------+--------------------------+-------------------------------------------------------------------------+
| Column            | Type                     | Modifiers                                                               |
|-------------------+--------------------------+-------------------------------------------------------------------------|
| id                | integer                  |  not null default nextval('sentry_organizationmember_id_seq'::regclass) |
| organization_id   | integer                  |  not null                                                               |
| user_id           | integer                  |                                                                         |
| type              | integer                  |  not null                                                               |
| date_added        | timestamp with time zone |  not null                                                               |
| email             | character varying(75)    |                                                                         |
| has_global_access | boolean                  |  not null                                                               |
| flags             | bigint                   |  not null                                                               |
| counter           | integer                  |                                                                         |
| role              | character varying(32)    |  not null                                                               |
+-------------------+--------------------------+-------------------------------------------------------------------------+

class OrganizationMember(Model):
    """
    Identifies relationships between teams and users.

    Users listed as team members are considered to have access to all projects
    and could be thought of as team owners (though their access level may not)
    be set to ownership.
    """
    __core__ = True

    organization = FlexibleForeignKey('sentry.Organization', related_name="member_set")

    user = FlexibleForeignKey(settings.AUTH_USER_MODEL, null=True, blank=True,
                             related_name="sentry_orgmember_set")
    email = models.EmailField(null=True, blank=True)
    role = models.CharField(
        choices=roles.get_choices(),
        max_length=32,
        default=roles.get_default().id,
    )
    flags = BitField(flags=(
        ('sso:linked', 'sso:linked'),
        ('sso:invalid', 'sso:invalid'),
    ), default=0)
    token = models.CharField(max_length=64, null=True, blank=True, unique=True)
    date_added = models.DateTimeField(default=timezone.now)
    has_global_access = models.BooleanField(default=True)
    teams = models.ManyToManyField('sentry.Team', blank=True,
                                   through='sentry.OrganizationMemberTeam')

    # Deprecated -- no longer used
    type = BoundedPositiveIntegerField(default=50, blank=True)

    class Meta:
        app_label = 'sentry'
        db_table = 'sentry_organizationmember'
        unique_together = (
            ('organization', 'user'),
            ('organization', 'email'),
        )

    __repr__ = sane_repr('organization_id', 'user_id', 'role',)

    @transaction.atomic
    def save(self, *args, **kwargs):
        assert self.user_id or self.email, \
            'Must set user or email'
        super(OrganizationMember, self).save(*args, **kwargs)

    @property
    def is_pending(self):
        return self.user_id is None

    @property
    def legacy_token(self):
        checksum = md5()
        checksum.update(six.text_type(self.organization_id).encode('utf-8'))
        checksum.update(self.get_email().encode('utf-8'))
        checksum.update(force_bytes(settings.SECRET_KEY))
        return checksum.hexdigest()

    def generate_token(self):
        return uuid4().hex + uuid4().hex

    def get_invite_link(self):
        if not self.is_pending:
            return None
        return absolute_uri(reverse('sentry-accept-invite', kwargs={
            'member_id': self.id,
            'token': self.token or self.legacy_token,
        }))

    def send_invite_email(self):
        from sentry.utils.email import MessageBuilder

        context = {
            'email': self.email,
            'organization': self.organization,
            'url': self.get_invite_link(),
        }

        msg = MessageBuilder(
            subject='Join %s in using Sentry' % self.organization.name,
            template='sentry/emails/member-invite.txt',
            html_template='sentry/emails/member-invite.html',
            type='organization.invite',
            context=context,
        )

        try:
            msg.send_async([self.get_email()])
        except Exception as e:
            logger = get_logger(name='sentry.mail')
            logger.exception(e)

    def send_sso_link_email(self):
        from sentry.utils.email import MessageBuilder

        context = {
            'email': self.email,
            'organization_name': self.organization.name,
            'url': absolute_uri(reverse('sentry-auth-organization', kwargs={
                'organization_slug': self.organization.slug,
            })),
        }

        msg = MessageBuilder(
            subject='Action Required for %s' % (self.organization.name,),
            template='sentry/emails/auth-link-identity.txt',
            html_template='sentry/emails/auth-link-identity.html',
            type='organization.auth_link',
            context=context,
        )
        msg.send_async([self.get_email()])

    def get_display_name(self):
        if self.user_id:
            return self.user.get_display_name()
        return self.email

    def get_label(self):
        if self.user_id:
            return self.user.get_label()
        return self.email or self.id

    def get_email(self):
        if self.user_id:
            return self.user.email
        return self.email

    def get_avatar_type(self):
        if self.user_id:
            return self.user.get_avatar_type()

    def get_audit_log_data(self):
        from sentry.models import Team
        return {
            'email': self.email,
            'user': self.user_id,
            'teams': list(Team.objects.filter(
                id__in=OrganizationMemberTeam.objects.filter(
                    organizationmember=self,
                    is_active=True,
                ).values_list('team', flat=True)
            )),
            'has_global_access': self.has_global_access,
            'role': self.role,
        }

    def get_teams(self):
        from sentry.models import Team

        if roles.get(self.role).is_global:
            return self.organization.team_set.all()

        return Team.objects.filter(
            id__in=OrganizationMemberTeam.objects.filter(
                organizationmember=self,
                is_active=True,
            ).values('team')
        )

    def get_scopes(self):
        return roles.get(self.role).scopes
*/