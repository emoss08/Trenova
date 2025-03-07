export type DatabaseBackup = {
  filename: string;
  size: number;
  createdAt: number;
  database: string;
  downloadUrl: string;
};

export type DatabaseBackupListResponse = {
  backups: DatabaseBackup[];
};
