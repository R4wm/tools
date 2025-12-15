#!/usr/bin/env python3
"""
Jira Management Script

Create, read, update, and manage Jira tickets from the command line.
Supports Jira Wiki Markup format for rich text formatting.
Config file: ~/.config/.jira_config.json

Usage:
    python3 jira_update.py --create --project LUC --summary "Title" --description "Details"
    python3 jira_update.py --read
    python3 jira_update.py --comment "Your comment here"
    python3 jira_update.py --ticket LUC-1234 --comment "Update for different ticket"
    python3 jira_update.py --format wiki --comment "h1. Title\\n\\nSome content"
"""

import argparse
import json
import os
import sys
import urllib.request
import urllib.error
from pathlib import Path
from base64 import b64encode


def load_config():
    """Load configuration from ~/.config/.jira_config.json"""
    config_path = Path.home() / ".config" / ".jira_config.json"

    if not config_path.exists():
        print(f"Config file not found: {config_path}")
        print("Please create the config file. See README.md for setup instructions.")
        return None

    try:
        with open(config_path, 'r') as f:
            config = json.load(f)
        return config
    except json.JSONDecodeError as e:
        print(f"Error parsing config file: {e}")
        return None
    except Exception as e:
        print(f"Error reading config file: {e}")
        return None


def get_config_value(config, key, args_value=None, env_var=None):
    """
    Get configuration value with priority:
    1. Command-line argument (highest priority)
    2. Config file
    3. Environment variable (fallback)
    """
    if args_value:
        return args_value

    if config and key in config:
        return config[key]

    if env_var and env_var in os.environ:
        return os.environ[env_var]

    return None


def create_jira_ticket(jira_url, project, summary, description, issue_type, email, api_token, story_points=None):
    """
    Create a new Jira ticket using the Jira REST API.

    Args:
        jira_url: Base Jira URL (e.g., https://edgecast.atlassian.net)
        project: Project key (e.g., LUC)
        summary: Ticket summary/title
        description: Ticket description text
        issue_type: Issue type (Task, Bug, Story, etc.)
        email: Jira account email
        api_token: Jira API token
        story_points: Story points (optional, default: None)

    Returns:
        Ticket key if successful, None otherwise
    """
    # Construct API endpoint
    api_url = f"{jira_url}/rest/api/2/issue"

    # Prepare the payload
    payload = {
        "fields": {
            "project": {
                "key": project
            },
            "summary": summary,
            "description": description,
            "issuetype": {
                "name": issue_type
            }
        }
    }

    # Add story points if provided (customfield_10032 is typically Story Points in Jira)
    if story_points is not None:
        payload["fields"]["customfield_10032"] = story_points

    # Prepare authentication (Basic Auth with email and API token)
    credentials = f"{email}:{api_token}"
    encoded_credentials = b64encode(credentials.encode('utf-8')).decode('ascii')

    # Prepare request
    data = json.dumps(payload).encode('utf-8')
    headers = {
        'Authorization': f'Basic {encoded_credentials}',
        'Content-Type': 'application/json',
        'Accept': 'application/json'
    }

    req = urllib.request.Request(api_url, data=data, headers=headers, method='POST')

    try:
        with urllib.request.urlopen(req) as response:
            response_data = response.read().decode('utf-8')
            result = json.loads(response_data)
            ticket_key = result.get('key')
            print(f"✅ Successfully created ticket {ticket_key}")
            print(f"   Summary: {summary}")
            print(f"   Type: {issue_type}")
            print(f"   Link: {jira_url}/browse/{ticket_key}")
            return ticket_key
    except urllib.error.HTTPError as e:
        error_body = e.read().decode('utf-8')
        print(f"❌ Failed to create ticket: HTTP {e.code}")
        print(f"   Error: {error_body}")
        return None
    except urllib.error.URLError as e:
        print(f"❌ Network error: {e.reason}")
        return None
    except Exception as e:
        print(f"❌ Unexpected error: {e}")
        return None


def update_jira_description(jira_url, ticket_id, description, email, api_token):
    """
    Update the description of a Jira ticket using the Jira REST API.

    Args:
        jira_url: Base Jira URL (e.g., https://edgecast.atlassian.net)
        ticket_id: Jira ticket ID (e.g., LUC-1385)
        description: New description text
        email: Jira account email
        api_token: Jira API token

    Returns:
        True if successful, False otherwise
    """
    # Construct API endpoint
    api_url = f"{jira_url}/rest/api/2/issue/{ticket_id}"

    # Prepare the description payload
    payload = {
        "fields": {
            "description": description
        }
    }

    # Prepare authentication (Basic Auth with email and API token)
    credentials = f"{email}:{api_token}"
    encoded_credentials = b64encode(credentials.encode('utf-8')).decode('ascii')

    # Prepare request
    data = json.dumps(payload).encode('utf-8')
    headers = {
        'Authorization': f'Basic {encoded_credentials}',
        'Content-Type': 'application/json',
        'Accept': 'application/json'
    }

    req = urllib.request.Request(api_url, data=data, headers=headers, method='PUT')

    try:
        with urllib.request.urlopen(req) as response:
            print(f"✅ Successfully updated description for {ticket_id}")
            print(f"   Link: {jira_url}/browse/{ticket_id}")
            return True
    except urllib.error.HTTPError as e:
        error_body = e.read().decode('utf-8')
        print(f"❌ Failed to update description: HTTP {e.code}")
        print(f"   Error: {error_body}")
        return False
    except urllib.error.URLError as e:
        print(f"❌ Network error: {e.reason}")
        return False
    except Exception as e:
        print(f"❌ Unexpected error: {e}")
        return False


def close_jira_ticket(jira_url, ticket_id, email, api_token):
    """
    Close a Jira ticket by transitioning it to Done/Closed status.

    Args:
        jira_url: Base Jira URL (e.g., https://edgecast.atlassian.net)
        ticket_id: Jira ticket ID (e.g., LUC-1385)
        email: Jira account email
        api_token: Jira API token

    Returns:
        True if successful, False otherwise
    """
    # First, get available transitions
    credentials = f"{email}:{api_token}"
    encoded_credentials = b64encode(credentials.encode('utf-8')).decode('ascii')

    headers = {
        'Authorization': f'Basic {encoded_credentials}',
        'Content-Type': 'application/json',
        'Accept': 'application/json'
    }

    # Get available transitions
    transitions_url = f"{jira_url}/rest/api/2/issue/{ticket_id}/transitions"

    try:
        req = urllib.request.Request(transitions_url, headers=headers, method='GET')
        with urllib.request.urlopen(req) as response:
            response_data = response.read().decode('utf-8')
            transitions = json.loads(response_data)

        # Find the "Done", "Closed", or "Resolved" transition
        close_transition = None
        for transition in transitions.get('transitions', []):
            name = transition.get('name', '').lower()
            if name in ['done', 'closed', 'close', 'resolved', 'resolve']:
                close_transition = transition
                break

        if not close_transition:
            print(f"❌ Could not find a close/done transition for {ticket_id}")
            print("   Available transitions:")
            for t in transitions.get('transitions', []):
                print(f"     - {t.get('name')}")
            return False

        # Execute the transition
        transition_id = close_transition['id']
        transition_name = close_transition['name']

        payload = {
            "transition": {
                "id": transition_id
            }
        }

        data = json.dumps(payload).encode('utf-8')
        req = urllib.request.Request(transitions_url, data=data, headers=headers, method='POST')

        with urllib.request.urlopen(req) as response:
            print(f"✅ Successfully closed {ticket_id} (transition: {transition_name})")
            print(f"   Link: {jira_url}/browse/{ticket_id}")
            return True

    except urllib.error.HTTPError as e:
        error_body = e.read().decode('utf-8')
        print(f"❌ Failed to close ticket: HTTP {e.code}")
        print(f"   Error: {error_body}")
        return False
    except urllib.error.URLError as e:
        print(f"❌ Network error: {e.reason}")
        return False
    except Exception as e:
        print(f"❌ Unexpected error: {e}")
        return False


def add_jira_comment(jira_url, ticket_id, comment, email, api_token):
    """
    Add a comment to a Jira ticket using the Jira REST API.

    Args:
        jira_url: Base Jira URL (e.g., https://edgecast.atlassian.net)
        ticket_id: Jira ticket ID (e.g., LUC-1385)
        comment: Comment text to add
        email: Jira account email
        api_token: Jira API token

    Returns:
        True if successful, False otherwise
    """
    # Construct API endpoint (try v2 first, it works for both Cloud and Server)
    api_url = f"{jira_url}/rest/api/2/issue/{ticket_id}/comment"

    # Prepare the comment payload (simple format for v2)
    payload = {
        "body": comment
    }

    # Prepare authentication (Basic Auth with email and API token)
    credentials = f"{email}:{api_token}"
    encoded_credentials = b64encode(credentials.encode('utf-8')).decode('ascii')

    # Prepare request
    data = json.dumps(payload).encode('utf-8')
    headers = {
        'Authorization': f'Basic {encoded_credentials}',
        'Content-Type': 'application/json',
        'Accept': 'application/json'
    }

    req = urllib.request.Request(api_url, data=data, headers=headers, method='POST')

    try:
        with urllib.request.urlopen(req) as response:
            response_data = response.read().decode('utf-8')
            result = json.loads(response_data)
            print(f"✅ Successfully added comment to {ticket_id}")
            print(f"   Comment ID: {result.get('id', 'unknown')}")
            print(f"   Link: {jira_url}/browse/{ticket_id}")
            return True
    except urllib.error.HTTPError as e:
        error_body = e.read().decode('utf-8')
        print(f"❌ Failed to add comment: HTTP {e.code}")
        print(f"   Error: {error_body}")
        return False
    except urllib.error.URLError as e:
        print(f"❌ Network error: {e.reason}")
        return False
    except Exception as e:
        print(f"❌ Unexpected error: {e}")
        return False


def read_jira_ticket(jira_url, ticket_id, email, api_token, show_comments=True, max_comments=3):
    """
    Read and display information from a Jira ticket using the Jira REST API.

    Args:
        jira_url: Base Jira URL (e.g., https://edgecast.atlassian.net)
        ticket_id: Jira ticket ID (e.g., LUC-1385)
        email: Jira account email
        api_token: Jira API token
        show_comments: Whether to display recent comments (default: True)
        max_comments: Maximum number of recent comments to show (default: 3)

    Returns:
        True if successful, False otherwise
    """
    # Construct API endpoint
    api_url = f"{jira_url}/rest/api/2/issue/{ticket_id}"

    # Prepare authentication
    credentials = f"{email}:{api_token}"
    encoded_credentials = b64encode(credentials.encode('utf-8')).decode('ascii')

    headers = {
        'Authorization': f'Basic {encoded_credentials}',
        'Accept': 'application/json'
    }

    req = urllib.request.Request(api_url, headers=headers, method='GET')

    try:
        with urllib.request.urlopen(req) as response:
            data = json.loads(response.read().decode('utf-8'))

            # Extract key information
            print("=" * 80)
            print(f"Ticket: {data['key']}")
            print(f"Summary: {data['fields']['summary']}")
            print(f"Status: {data['fields']['status']['name']}")

            # Display assignee
            if data['fields'].get('assignee'):
                print(f"Assignee: {data['fields']['assignee']['displayName']}")
            else:
                print("Assignee: Unassigned")

            # Display reporter
            if data['fields'].get('reporter'):
                print(f"Reporter: {data['fields']['reporter']['displayName']}")

            # Display priority if available
            if data['fields'].get('priority'):
                print(f"Priority: {data['fields']['priority']['name']}")

            # Display created and updated dates
            if data['fields'].get('created'):
                print(f"Created: {data['fields']['created']}")
            if data['fields'].get('updated'):
                print(f"Updated: {data['fields']['updated']}")

            print("=" * 80)

            # Display description
            print("\nDescription:")
            desc = data['fields'].get('description', 'No description')
            print(desc if desc else 'No description')
            print("\n" + "=" * 80)

            # Display recent comments if requested
            if show_comments and 'comment' in data['fields'] and data['fields']['comment']['comments']:
                comments = data['fields']['comment']['comments']
                print(f"\nRecent Comments ({len(comments)} total, showing last {min(max_comments, len(comments))}):")
                for comment in comments[-max_comments:]:
                    print(f"\n[{comment['author']['displayName']}] {comment['created']}")
                    print(comment['body'])
                print("\n" + "=" * 80)

            # Display link
            print(f"\nLink: {jira_url}/browse/{ticket_id}")

            return True

    except urllib.error.HTTPError as e:
        error_body = e.read().decode('utf-8')
        print(f"❌ Failed to read ticket: HTTP {e.code}")
        print(f"   Error: {error_body}")
        return False
    except urllib.error.URLError as e:
        print(f"❌ Network error: {e.reason}")
        return False
    except Exception as e:
        print(f"❌ Unexpected error: {e}")
        return False


def search_jira_tickets(jira_url, jql, email, api_token, max_results=50):
    """
    Search for Jira tickets using JQL (Jira Query Language).

    Args:
        jira_url: Base Jira URL (e.g., https://edgecast.atlassian.net)
        jql: JQL query string
        email: Jira account email
        api_token: Jira API token
        max_results: Maximum number of results to return (default: 50)

    Returns:
        List of ticket data if successful, None otherwise
    """
    # Construct API endpoint (v3 for search/jql)
    api_url = f"{jira_url}/rest/api/3/search/jql"

    # URL encode the JQL query
    import urllib.parse
    params = urllib.parse.urlencode({
        'jql': jql,
        'maxResults': max_results,
        'fields': 'summary,status,assignee,reporter,created,updated,description'
    })

    full_url = f"{api_url}?{params}"

    # Prepare authentication
    credentials = f"{email}:{api_token}"
    encoded_credentials = b64encode(credentials.encode('utf-8')).decode('ascii')

    headers = {
        'Authorization': f'Basic {encoded_credentials}',
        'Accept': 'application/json'
    }

    req = urllib.request.Request(full_url, headers=headers, method='GET')

    try:
        with urllib.request.urlopen(req) as response:
            data = json.loads(response.read().decode('utf-8'))

            issues = data.get('issues', [])
            total = data.get('total', 0)

            print("=" * 80)
            print(f"Found {total} ticket(s) (showing up to {max_results})")
            print("=" * 80)
            print()

            if not issues:
                print("No tickets found.")
                return []

            for issue in issues:
                key = issue['key']
                fields = issue['fields']
                summary = fields.get('summary', 'No summary')
                status = fields['status']['name']

                assignee = 'Unassigned'
                if fields.get('assignee'):
                    assignee = fields['assignee']['displayName']

                reporter = 'Unknown'
                if fields.get('reporter'):
                    reporter = fields['reporter']['displayName']

                created = fields.get('created', 'Unknown')[:10]  # Just the date
                updated = fields.get('updated', 'Unknown')[:10]

                print(f"🎫 {key}: {summary}")
                print(f"   Status: {status}")
                print(f"   Assignee: {assignee} | Reporter: {reporter}")
                print(f"   Created: {created} | Updated: {updated}")
                print(f"   Link: {jira_url}/browse/{key}")
                print()

            return issues

    except urllib.error.HTTPError as e:
        error_body = e.read().decode('utf-8')
        print(f"❌ Failed to search tickets: HTTP {e.code}")
        print(f"   Error: {error_body}")
        return None
    except urllib.error.URLError as e:
        print(f"❌ Network error: {e.reason}")
        return None
    except Exception as e:
        print(f"❌ Unexpected error: {e}")
        return None


def main():
    parser = argparse.ArgumentParser(
        description='Create, read, update, and manage Jira tickets from the command line',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Configuration priority (highest to lowest):
  1. Command-line arguments
  2. ~/.config/.jira_config.json
  3. Environment variables

Format Options:
  wiki (default) - Use Jira Wiki Markup (h1., h2., {code}, *bold*, etc.)
  text           - Plain text format

Examples:
  # Create a new ticket
  python3 jira_update.py --create --project LUC --summary "Fix login bug" --description "Users cannot login" --issue-type Bug

  # Search for tickets (using JQL)
  python3 jira_update.py --search "project = TVM AND assignee = currentUser() AND status != Done"
  python3 jira_update.py --search "project = TVM AND reporter = currentUser() ORDER BY updated DESC"
  python3 jira_update.py --search "text ~ 'pgdog' OR text ~ '40001'" --max-results 20

  # Read tickets
  python3 jira_update.py --read
  python3 jira_update.py --ticket LUC-1234 --read

  # Add comments
  python3 jira_update.py --comment "Feature completed"
  python3 jira_update.py --ticket LUC-1234 --comment "Status update"

  # Update description
  python3 jira_update.py --description "New description for ticket"

  # Close ticket
  python3 jira_update.py --close
  python3 jira_update.py --comment "Final update" --close

  # Use Wiki markup
  python3 jira_update.py --format wiki --comment "h1. Title\\n\\nSome *bold* text"
  python3 jira_update.py --format text --comment "Plain text comment"

  # Create and immediately add a comment
  python3 jira_update.py --create --project LUC --summary "New feature" --description "Feature description" --comment "Started work on this"
        """
    )

    parser.add_argument(
        '--ticket',
        '-t',
        help='Jira ticket ID (e.g., LUC-1385)',
        type=str
    )

    parser.add_argument(
        '--comment',
        '-c',
        help='Comment text to add to the ticket',
        type=str
    )

    parser.add_argument(
        '--description',
        '-d',
        help='New description for the ticket',
        type=str
    )

    parser.add_argument(
        '--read',
        '-r',
        action='store_true',
        help='Read and display ticket information'
    )

    parser.add_argument(
        '--close',
        action='store_true',
        help='Close the ticket (transition to Done/Closed status)'
    )

    parser.add_argument(
        '--jira-url',
        '-u',
        help='Jira base URL (e.g., https://edgecast.atlassian.net)',
        type=str
    )

    parser.add_argument(
        '--email',
        '-e',
        help='Jira account email',
        type=str
    )

    parser.add_argument(
        '--api-token',
        '-a',
        help='Jira API token',
        type=str
    )

    parser.add_argument(
        '--format',
        '-f',
        help='Content format: "wiki" for Jira Wiki Markup, "text" for plain text (default: wiki)',
        type=str,
        choices=['wiki', 'text'],
        default=None
    )

    parser.add_argument(
        '--create',
        action='store_true',
        help='Create a new Jira ticket'
    )

    parser.add_argument(
        '--summary',
        '-s',
        help='Ticket summary/title (required for --create)',
        type=str
    )

    parser.add_argument(
        '--project',
        '-p',
        help='Project key (e.g., LUC) (required for --create)',
        type=str
    )

    parser.add_argument(
        '--issue-type',
        '-i',
        help='Issue type: Task, Bug, Story, etc. (default: Task)',
        type=str,
        default='Task'
    )

    parser.add_argument(
        '--story-points',
        help='Story points (optional, typically 1-13)',
        type=int
    )

    parser.add_argument(
        '--search',
        help='Search tickets using JQL (Jira Query Language)',
        type=str
    )

    parser.add_argument(
        '--max-results',
        help='Maximum number of search results (default: 50)',
        type=int,
        default=50
    )

    args = parser.parse_args()

    # Validate that at least one action is provided
    if not args.comment and not args.description and not args.close and not args.read and not args.create and not args.search:
        print("❌ Error: You must provide at least one action: --create, --read, --search, --comment, --description, or --close")
        parser.print_help()
        sys.exit(1)

    # Validate create-specific requirements
    if args.create:
        if not args.summary:
            print("❌ Error: --summary is required when using --create")
            sys.exit(1)
        if not args.project:
            print("❌ Error: --project is required when using --create")
            sys.exit(1)
        if not args.description:
            print("❌ Error: --description is required when using --create")
            sys.exit(1)

    # Load config file
    config = load_config()
    if config is None and not all([args.jira_url, args.email, args.api_token]):
        print("\n❌ Config file not found and not all required arguments provided.")
        print("Either create ~/.config/.jira_config.json or provide all required arguments.")
        sys.exit(1)

    # Get configuration values with priority
    ticket_id = get_config_value(config, 'ticket_id', args.ticket, 'JIRA_TICKET_ID')
    jira_url = get_config_value(config, 'jira_url', args.jira_url, 'JIRA_URL')
    email = get_config_value(config, 'email', args.email, 'JIRA_EMAIL')
    api_token = get_config_value(config, 'api_token', args.api_token, 'JIRA_API_TOKEN')
    content_format = get_config_value(config, 'format', args.format, 'JIRA_FORMAT') or 'wiki'
    project = get_config_value(config, 'project', args.project, 'JIRA_PROJECT')

    # Validate required fields (ticket_id not required for --create or --search)
    missing_fields = []
    if not args.create and not args.search and not ticket_id:
        missing_fields.append('ticket_id (--ticket or config)')
    if not jira_url:
        missing_fields.append('jira_url (--jira-url or config)')
    if not email:
        missing_fields.append('email (--email or config)')
    if not api_token:
        missing_fields.append('api_token (--api-token or config)')

    if missing_fields:
        print("❌ Missing required configuration:")
        for field in missing_fields:
            print(f"   - {field}")
        sys.exit(1)

    # Clean up jira_url (remove trailing slash)
    jira_url = jira_url.rstrip('/')

    all_success = True

    # Search for tickets if requested
    if args.search:
        print(f"🔍 Searching for tickets...")
        print(f"   Jira: {jira_url}")
        print(f"   JQL: {args.search}")
        print()
        results = search_jira_tickets(jira_url, args.search, email, api_token, args.max_results)
        if results is None:
            all_success = False
            sys.exit(1)

    # Create ticket if requested
    if args.create:
        print(f"🆕 Creating new ticket in project {project}...")
        print(f"   Jira: {jira_url}")
        print(f"   Format: {content_format.upper()}")
        ticket_key = create_jira_ticket(
            jira_url,
            project,
            args.summary,
            args.description,
            args.issue_type,
            email,
            api_token,
            args.story_points
        )
        if ticket_key:
            # Set ticket_id for any subsequent operations
            ticket_id = ticket_key
        else:
            all_success = False
            sys.exit(1)

    # Read ticket if requested (this is a read-only operation)
    if args.read:
        print(f"📖 Reading {ticket_id}...")
        print(f"   Jira: {jira_url}")
        success = read_jira_ticket(jira_url, ticket_id, email, api_token)
        all_success = all_success and success

    # For update operations, show the format
    if args.comment or args.description or args.close:
        print(f"🔄 Updating {ticket_id}...")
        print(f"   Jira: {jira_url}")
        print(f"   Format: {content_format.upper()}")

        # Note: We always send content as-is to Jira. When format='wiki', content should use
        # Jira Wiki Markup syntax (h1., h2., {code}, etc.). When format='text', content is plain text.

        # Update description if provided
        if args.description:
            print(f"   📝 Updating description...")
            success = update_jira_description(jira_url, ticket_id, args.description, email, api_token)
            all_success = all_success and success

        # Add comment if provided
        if args.comment:
            print(f"   💬 Adding comment...")
            success = add_jira_comment(jira_url, ticket_id, args.comment, email, api_token)
            all_success = all_success and success

        # Close ticket if requested
        if args.close:
            print(f"   🔒 Closing ticket...")
            success = close_jira_ticket(jira_url, ticket_id, email, api_token)
            all_success = all_success and success

    if all_success:
        sys.exit(0)
    else:
        sys.exit(1)


if __name__ == "__main__":
    main()
