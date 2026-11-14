-- An organization's icon is its square mark, drawn where a logo has no room: the sidebar,
-- the organization switcher. The logo stays the full mark its documents and emails carry.
ALTER TABLE "organizations" ADD COLUMN "icon_url" varchar(255);
