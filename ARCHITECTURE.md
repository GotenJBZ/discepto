# Moderation
Discepto uses a complex Role Based Access Control (RBAC).
Each user can have one or multiple roles.
Each role gives a set of permissions.
When retrieving roles, permissions get summed (boolean OR): the permissions set to true always win.
Be sure to see the first database migration in the folder /migrations to see how this is implemented
in the database

## Global roles
They override any local role. The first user registered on Discepto has complete control over
the platform (meaning it gets a global role with full permissions).

## Local roles
They are local to a subdiscepto. Discepto has some preset local roles, but also supports the
creations of custom ones.

# Security
In code, security and roles are enforced following in part the design guidelines of the object-capability-model.

Inside the discepto codebase, there is pure data (inside the "domain" package) and data handlers (inside the "db" package).

To retrieve some data or delete it, or [action on the data], you must have a reference (handler) to it (object-capability-model).

## Data handlers
Handlers manage access to the underlying data stored in the database.

### Applied example: Delete an essay
To delete an essay, you must have an handler to it.
You can get an essay handler (EssayH) ONLY from a subdiscepto handler (SubdisceptoH).
You can get a SubdisceptoH ONLY if you are at least a member of the corresponding subdiscepto.
Obviously, when trying to execute actions on an handler the corresponding permissions are checked.
As you can see, using handlers makes dealing with this security stuff a lot easier and safer than creating a lot of public functions to directly write data to the database. Handlers are a lot harder to misuse.
