package cli

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/auth0/go-auth0/management"
	"github.com/spf13/cobra"

	"github.com/auth0/auth0-cli/internal/ansi"
	"github.com/auth0/auth0-cli/internal/auth0"
	"github.com/auth0/auth0-cli/internal/prompt"
	"github.com/auth0/auth0-cli/internal/users"
)

var (
	userID = Argument{
		Name: "User ID",
		Help: "Id of the user.",
	}

	userConnection = Flag{
		Name:       "Connection",
		LongForm:   "connection",
		ShortForm:  "c",
		Help:       "Name of the database connection this user should be created in.",
		IsRequired: true,
	}
	userEmail = Flag{
		Name:       "Email",
		LongForm:   "email",
		ShortForm:  "e",
		Help:       "The user's email.",
		IsRequired: true,
	}
	userPassword = Flag{
		Name:       "Password",
		LongForm:   "password",
		ShortForm:  "p",
		Help:       "Initial password for this user (mandatory for non-SMS connections).",
		IsRequired: true,
	}
	userUsername = Flag{
		Name:      "Username",
		LongForm:  "username",
		ShortForm: "u",
		Help:      "The user's username. Only valid if the connection requires a username.",
	}
	userName = Flag{
		Name:         "Name",
		LongForm:     "name",
		ShortForm:    "n",
		Help:         "The user's full name.",
		IsRequired:   true,
		AlwaysPrompt: true,
	}
	userQuery = Flag{
		Name:       "Query",
		LongForm:   "query",
		ShortForm:  "q",
		Help:       "Query in Lucene query syntax. See https://auth0.com/docs/users/user-search/user-search-query-syntax for more details.",
		IsRequired: true,
	}
	userSort = Flag{
		Name:      "Sort",
		LongForm:  "sort",
		ShortForm: "s",
		Help:      "Field to sort by. Use 'field:order' where 'order' is '1' for ascending and '-1' for descending. e.g. 'created_at:1'.",
	}
	userNumber = Flag{
		Name:      "Number",
		LongForm:  "number",
		ShortForm: "n",
		Help:      "Number of users, that match the search criteria, to retrieve. Minimum 1, maximum 1000. If limit is hit, refine the search query.",
	}
	userImportTemplate = Flag{
		Name:       "Template",
		LongForm:   "template",
		ShortForm:  "t",
		Help:       "Name of JSON example to be used.",
		IsRequired: false,
	}
	userImportTemplateBody = Flag{
		Name:       "Template Body",
		LongForm:   "template-body",
		ShortForm:  "b",
		Help:       "JSON template body that contains an array of user(s) to be imported.",
		IsRequired: false,
	}
	userEmailResults = Flag{
		Name:       "Email Completion Results",
		LongForm:   "email-results",
		ShortForm:  "r",
		Help:       "When true, sends a completion email to all tenant owners when the job is finished. The default is true, so you must explicitly set this parameter to false if you do not want emails sent.",
		IsRequired: false,
	}
	userImportUpsert = Flag{
		Name:       "Upsert",
		LongForm:   "upsert",
		ShortForm:  "u",
		Help:       "When set to false, pre-existing users that match on email address, user ID, or username will fail. When set to true, pre-existing users that match on any of these fields will be updated, but only with upsertable attributes.",
		IsRequired: false,
	}
	userImportOptions = pickerOptions{
		{"Empty", users.EmptyExample},
		{"Basic Example", users.BasicExample},
		{"Custom Password Hash Example", users.CustomPasswordHashExample},
		{"MFA Factors Example", users.MFAFactors},
	}
)

func usersCmd(cli *cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Manage resources for users",
		Long:  "Manage resources for users.",
	}

	cmd.SetUsageTemplate(resourceUsageTemplate())
	cmd.AddCommand(searchUsersCmd(cli))
	cmd.AddCommand(createUserCmd(cli))
	cmd.AddCommand(showUserCmd(cli))
	cmd.AddCommand(updateUserCmd(cli))
	cmd.AddCommand(deleteUserCmd(cli))
	cmd.AddCommand(userRolesCmd(cli))
	cmd.AddCommand(openUserCmd(cli))
	cmd.AddCommand(userBlocksCmd(cli))
	cmd.AddCommand(importUsersCmd(cli))

	return cmd
}

func searchUsersCmd(cli *cli) *cobra.Command {
	var inputs struct {
		query  string
		sort   string
		number int
	}

	cmd := &cobra.Command{
		Use:   "search",
		Args:  cobra.NoArgs,
		Short: "Search for users",
		Long:  "Search for users. To create one, run: `auth0 users create`.",
		Example: `  auth0 users search
  auth0 users search --query user_id:"<user-id>"
  auth0 users search --query name:"Bob" --sort "name:1"
  auth0 users search -q name:"Bob" -s "name:1" --number 200
  auth0 users search -q name:"Bob" -s "name:1" -n 200 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := userQuery.Ask(cmd, &inputs.query, nil); err != nil {
				return err
			}

			queryParams := []management.RequestOption{
				management.Query(inputs.query),
			}
			if inputs.sort != "" {
				queryParams = append(queryParams, management.Parameter("sort", inputs.sort))
			}

			if inputs.number < 1 || inputs.number > 1000 {
				return fmt.Errorf("number flag invalid, please pass a number between 1 and 1000")
			}

			list, err := getWithPagination(
				cmd.Context(),
				inputs.number,
				func(opts ...management.RequestOption) (result []interface{}, hasNext bool, err error) {
					opts = append(opts, queryParams...)

					userList, err := cli.api.User.Search(opts...)
					if err != nil {
						return nil, false, err
					}

					var output []interface{}
					for _, user := range userList.Users {
						output = append(output, user)
					}

					return output, userList.HasNext(), nil
				},
			)
			if err != nil {
				return fmt.Errorf("failed to search for users: %w", err)
			}

			var foundUsers []*management.User
			for _, item := range list {
				foundUsers = append(foundUsers, item.(*management.User))
			}

			cli.renderer.UserSearch(foundUsers)

			return nil
		},
	}

	cmd.Flags().BoolVar(&cli.json, "json", false, "Output in json format.")
	userQuery.RegisterString(cmd, &inputs.query, "")
	userSort.RegisterString(cmd, &inputs.sort, "")
	userNumber.RegisterInt(cmd, &inputs.number, defaultPageSize)

	return cmd
}

func createUserCmd(cli *cli) *cobra.Command {
	var inputs struct {
		Connection string
		Email      string
		Password   string
		Username   string
		Name       string
	}

	cmd := &cobra.Command{
		Use:   "create",
		Args:  cobra.NoArgs,
		Short: "Create a new user",
		Long: "Create a new user.\n\n" +
			"To create interactively, use `auth0 users create` with no flags.\n\n" +
			"To create non-interactively, supply the name and other information through the available flags.",
		Example: `  auth0 users create 
  auth0 users create --name "John Doe" 
  auth0 users create --name "John Doe" --email john@example.com
  auth0 users create --name "John Doe" --email john@example.com --connection "Username-Password-Authentication" --username "example"
  auth0 users create -n "John Doe" -e john@example.com -c "Username-Password-Authentication" -u "example" --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Select from the available connection types
			// Users API currently support  database connections
			if err := userConnection.Select(cmd, &inputs.Connection, cli.connectionPickerOptions(), nil); err != nil {
				return err
			}

			// Prompt for user's name
			if err := userName.Ask(cmd, &inputs.Name, nil); err != nil {
				return err
			}

			// Prompt for user email
			if err := userEmail.Ask(cmd, &inputs.Email, nil); err != nil {
				return err
			}

			// //Prompt for user password
			if err := userPassword.AskPassword(cmd, &inputs.Password, nil); err != nil {
				return err
			}

			// The getConnReqUsername returns the value for the requires_username field for the selected connection
			// The result will be used to determine whether to prompt for username
			conn := cli.getConnReqUsername(auth0.StringValue(&inputs.Connection))
			requireUsername := auth0.BoolValue(conn)

			// Prompt for username if the requireUsername is set to true
			// Load values including the username's field into a fresh users instance
			a := &management.User{
				Connection: &inputs.Connection,
				Email:      &inputs.Email,
				Name:       &inputs.Name,
				Password:   &inputs.Password,
			}

			if requireUsername {
				if err := userUsername.Ask(cmd, &inputs.Username, nil); err != nil {
					return err
				}
				a.Username = &inputs.Username
			}
			// Create app
			if err := ansi.Waiting(func() error {
				return cli.api.User.Create(a)
			}); err != nil {
				return fmt.Errorf("Unable to create user: %w", err)
			}

			// Render Result
			cli.renderer.UserCreate(a, requireUsername)

			return nil
		},
	}

	cmd.Flags().BoolVar(&cli.json, "json", false, "Output in json format.")
	userName.RegisterString(cmd, &inputs.Name, "")
	userConnection.RegisterString(cmd, &inputs.Connection, "")
	userPassword.RegisterString(cmd, &inputs.Password, "")
	userEmail.RegisterString(cmd, &inputs.Email, "")
	userUsername.RegisterString(cmd, &inputs.Username, "")

	return cmd
}

func showUserCmd(cli *cli) *cobra.Command {
	var inputs struct {
		ID string
	}

	cmd := &cobra.Command{
		Use:   "show",
		Args:  cobra.MaximumNArgs(1),
		Short: "Show an existing user",
		Long:  "Display information about an existing user.",
		Example: `  auth0 users show 
  auth0 users show <user-id>
  auth0 users show <user-id> --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				if err := userID.Ask(cmd, &inputs.ID); err != nil {
					return err
				}
			} else {
				inputs.ID = args[0]
			}

			a := &management.User{ID: &inputs.ID}

			if err := ansi.Waiting(func() error {
				var err error
				a, err = cli.api.User.Read(inputs.ID)
				return err
			}); err != nil {
				return fmt.Errorf("Unable to load user: %w", err)
			}

			// get the current connection
			conn := stringSliceToCommaSeparatedString(cli.getUserConnection(a))
			a.Connection = auth0.String(conn)

			// parse the connection name to get the requireUsername status
			u := cli.getConnReqUsername(auth0.StringValue(a.Connection))
			requireUsername := auth0.BoolValue(u)

			cli.renderer.UserShow(a, requireUsername)
			return nil
		},
	}

	cmd.Flags().BoolVar(&cli.json, "json", false, "Output in json format.")

	return cmd
}

func deleteUserCmd(cli *cli) *cobra.Command {
	var inputs struct {
		ID string
	}

	cmd := &cobra.Command{
		Use:     "delete",
		Aliases: []string{"rm"},
		Args:    cobra.MaximumNArgs(1),
		Short:   "Delete a user",
		Long: "Delete a user.\n\n" +
			"To delete interactively, use `auth0 users delete` with no arguments.\n\n" +
			"To delete non-interactively, supply the user id and the `--force` flag to skip confirmation.",
		Example: `  auth0 users delete 
  auth0 users rm
  auth0 users delete <user-id>
  auth0 users delete <user-id> --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				if err := userID.Ask(cmd, &inputs.ID); err != nil {
					return err
				}
			} else {
				inputs.ID = args[0]
			}

			if !cli.force && canPrompt(cmd) {
				if confirmed := prompt.Confirm("Are you sure you want to proceed?"); !confirmed {
					return nil
				}
			}

			return ansi.Spinner("Deleting user", func() error {
				_, err := cli.api.User.Read(inputs.ID)

				if err != nil {
					return fmt.Errorf("Unable to delete user: %w", err)
				}

				return cli.api.User.Delete(inputs.ID)
			})
		},
	}

	cmd.Flags().BoolVar(&cli.force, "force", false, "Skip confirmation.")

	return cmd
}

func updateUserCmd(cli *cli) *cobra.Command {
	var inputs struct {
		ID         string
		Email      string
		Password   string
		Name       string
		Connection string
	}

	cmd := &cobra.Command{
		Use:   "update",
		Args:  cobra.MaximumNArgs(1),
		Short: "Update a user",
		Long: "Update a user.\n\n" +
			"To update interactively, use `auth0 users update` with no arguments.\n\n" +
			"To update non-interactively, supply the user id and other information through the available flags.",
		Example: `  auth0 users update 
  auth0 users update <user-id> 
  auth0 users update <user-id> --name "John Doe"
  auth0 users update <user-id> --name "John Doe" --email john.doe@example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				if err := userID.Ask(cmd, &inputs.ID); err != nil {
					return err
				}
			} else {
				inputs.ID = args[0]
			}

			var current *management.User

			if err := ansi.Waiting(func() error {
				var err error
				current, err = cli.api.User.Read(inputs.ID)
				return err
			}); err != nil {
				return fmt.Errorf("Unable to load user: %w", err)
			}
			// using getUserConnection to get connection name from user Identities
			// just using current.connection will return empty
			conn := stringSliceToCommaSeparatedString(cli.getUserConnection(current))
			current.Connection = auth0.String(conn)

			if err := userName.AskU(cmd, &inputs.Name, current.Name); err != nil {
				return err
			}

			if err := userEmail.AskU(cmd, &inputs.Email, current.Email); err != nil {
				return err
			}

			if err := userPassword.AskPasswordU(cmd, &inputs.Password, current.Password); err != nil {
				return err
			}

			// username cannot be updated for database connections
			// if err := userUsername.AskU(cmd, &inputs.Username, current.Username); err != nil {
			//	return err
			// }

			user := &management.User{}

			if len(inputs.Name) == 0 {
				user.Name = current.Name
			} else {
				user.Name = &inputs.Name
			}

			if len(inputs.Email) == 0 {
				user.Email = current.Email
			} else {
				user.Email = &inputs.Email
			}

			if len(inputs.Password) == 0 {
				user.Password = current.Password
			} else {
				user.Password = &inputs.Password
			}

			if len(inputs.Connection) == 0 {
				user.Connection = current.Connection
			} else {
				user.Connection = &inputs.Connection
			}

			if err := ansi.Waiting(func() error {
				return cli.api.User.Update(current.GetID(), user)
			}); err != nil {
				return fmt.Errorf("An unexpected error occurred while trying to update an user with Id '%s': %w", inputs.ID, err)
			}

			con := cli.getConnReqUsername(auth0.StringValue(user.Connection))
			requireUsername := auth0.BoolValue(con)

			cli.renderer.UserUpdate(user, requireUsername)
			return nil
		},
	}

	cmd.Flags().BoolVar(&cli.json, "json", false, "Output in json format.")
	userName.RegisterStringU(cmd, &inputs.Name, "")
	userConnection.RegisterStringU(cmd, &inputs.Connection, "")
	userPassword.RegisterStringU(cmd, &inputs.Password, "")
	userEmail.RegisterStringU(cmd, &inputs.Email, "")

	return cmd
}

func openUserCmd(cli *cli) *cobra.Command {
	var inputs struct {
		ID string
	}

	cmd := &cobra.Command{
		Use:   "open",
		Args:  cobra.MaximumNArgs(1),
		Short: "Open the user's settings page",
		Long:  "Open the settings page of a user in the Auth0 Dashboard.",
		Example: `  auth0 users open <id>
  auth0 users open "auth0|xxxxxxxxxx"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				if err := userID.Ask(cmd, &inputs.ID); err != nil {
					return err
				}
			} else {
				inputs.ID = args[0]
			}

			openManageURL(cli, cli.config.DefaultTenant, formatUserDetailsPath(url.PathEscape(inputs.ID)))
			return nil
		},
	}

	return cmd
}

func importUsersCmd(cli *cli) *cobra.Command {
	var inputs struct {
		Connection          string
		ConnectionId        string
		Template            string
		TemplateBody        string
		Upsert              bool
		SendCompletionEmail bool
	}
	cmd := &cobra.Command{
		Use:   "import",
		Args:  cobra.NoArgs,
		Short: "Import users from schema",
		Long: `Import users from schema. Issues a Create Import Users Job. 
The file size limit for a bulk import is 500KB. You will need to start multiple imports if your data exceeds this size.`,
		Example: `  auth0 users import
  auth0 users import --connection "Username-Password-Authentication"
  auth0 users import -c "Username-Password-Authentication" --template "Basic Example"
  auth0 users import -c "Username-Password-Authentication" -t "Basic Example" --upsert true
  auth0 users import -c "Username-Password-Authentication" -t "Basic Example" --upsert true --email-results false`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Select from the available connection types
			// Users API currently support database connections
			if err := userConnection.Select(cmd, &inputs.Connection, cli.connectionPickerOptions(), nil); err != nil {
				return err
			}

			// Get Connection ID
			conn, connErr := cli.api.Connection.ReadByName(inputs.Connection)
			if connErr != nil {
				return fmt.Errorf("Connection does not exist: %w", connErr)
			} else {
				inputs.ConnectionId = *conn.ID
			}

			// Present user with template options
			if templateErr := userImportTemplate.Select(cmd, &inputs.Template, userImportOptions.labels(), nil); templateErr != nil {
				return templateErr
			}

			editorErr := userImportTemplateBody.OpenEditor(
				cmd,
				&inputs.TemplateBody,
				userImportOptions.getValue(inputs.Template),
				inputs.Template+".*.json",
				cli.userImportEditorHint,
			)
			if editorErr != nil {
				return fmt.Errorf("Failed to capture input from the editor: %w", editorErr)
			}

			var confirmed bool
			if confirmedErr := prompt.AskBool("Do you want to import these user(s)?", &confirmed, true); confirmedErr != nil {
				return fmt.Errorf("Failed to capture prompt input: %w", confirmedErr)
			}

			if !confirmed {
				return nil
			}

			// Convert json array to map
			jsonstr := inputs.TemplateBody
			var jsonmap []map[string]interface{}
			jsonErr := json.Unmarshal([]byte(jsonstr), &jsonmap)
			if jsonErr != nil {
				return fmt.Errorf("Invalid JSON input: %w", jsonErr)
			}

			err := ansi.Waiting(func() error {
				return cli.api.Jobs.ImportUsers(&management.Job{
					ConnectionID:        &inputs.ConnectionId,
					Users:               jsonmap,
					Upsert:              &inputs.Upsert,
					SendCompletionEmail: &inputs.SendCompletionEmail,
				})
			})
			if err != nil {
				return err
			}

			cli.renderer.Heading("starting user import job...")
			fmt.Println(jsonstr)

			if inputs.SendCompletionEmail {
				cli.renderer.Infof("Results of your user import job will be sent to your email.")
			}

			return nil
		},
	}

	userConnection.RegisterString(cmd, &inputs.Connection, "")
	userImportTemplate.RegisterString(cmd, &inputs.Template, "")
	userEmailResults.RegisterBool(cmd, &inputs.SendCompletionEmail, true)
	userImportUpsert.RegisterBool(cmd, &inputs.Upsert, false)

	return cmd
}

func formatUserDetailsPath(id string) string {
	if len(id) == 0 {
		return ""
	}
	return fmt.Sprintf("users/%s", id)
}

func (c *cli) connectionPickerOptions() []string {
	var res []string

	list, err := c.api.Connection.List()
	if err != nil {
		fmt.Println(err)
	}
	for _, conn := range list.Connections {
		if conn.GetStrategy() == "auth0" {
			res = append(res, conn.GetName())
		}
	}

	return res
}

func (c *cli) getUserConnection(users *management.User) []string {
	var res []string
	for _, i := range users.Identities {
		res = append(res, fmt.Sprintf("%v", auth0.StringValue(i.Connection)))
	}

	return res
}

// This is a workaround to get the requires_username field nested inside Options field.
func (c *cli) getConnReqUsername(s string) *bool {
	conn, err := c.api.Connection.ReadByName(s)
	if err != nil {
		fmt.Println(err)
	}
	res := fmt.Sprintln(conn.Options)

	opts := &management.ConnectionOptions{}
	if err := json.Unmarshal([]byte(res), &opts); err != nil {
		fmt.Println(err)
	}

	return opts.RequiresUsername
}

func (c *cli) userImportEditorHint() {
	c.renderer.Infof("%s Once you close the editor, the user(s) will be imported. To cancel, CTRL+C.", ansi.Faint("Hint:"))
}
