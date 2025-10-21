/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */

export interface Container {
  Id: string;
  Names: string[];
  Image: string;
  ImageID: string;
  Command: string;
  Created: number;
  Ports: Port[];
  Labels: Record<string, string>;
  State: string;
  Status: string;
  HostConfig: {
    NetworkMode: string;
  };
  NetworkSettings: {
    Networks: Record<string, any>;
  };
  Mounts: Mount[];
  SizeRw?: number;
  SizeRootFs?: number;
}

export interface Port {
  IP?: string;
  PrivatePort: number;
  PublicPort?: number;
  Type: string;
}

export interface Mount {
  Type: string;
  Name?: string;
  Source: string;
  Destination: string;
  Mode: string;
  RW: boolean;
  Propagation: string;
}

export interface ContainerInspect {
  Id: string;
  Created: string;
  Path: string;
  Args: string[];
  State: ContainerState;
  Image: string;
  Name: string;
  RestartCount: number;
  Driver: string;
  Platform: string;
  MountLabel: string;
  ProcessLabel: string;
  AppArmorProfile: string;
  ExecIDs?: string[];
  HostConfig: any;
  GraphDriver: any;
  Mounts: Mount[];
  Config: ContainerConfig;
  NetworkSettings: any;
}

export interface ContainerState {
  Status: string;
  Running: boolean;
  Paused: boolean;
  Restarting: boolean;
  OOMKilled: boolean;
  Dead: boolean;
  Pid: number;
  ExitCode: number;
  Error: string;
  StartedAt: string;
  FinishedAt: string;
}

export interface ContainerConfig {
  Hostname: string;
  Domainname: string;
  User: string;
  AttachStdin: boolean;
  AttachStdout: boolean;
  AttachStderr: boolean;
  ExposedPorts?: Record<string, any>;
  Tty: boolean;
  OpenStdin: boolean;
  StdinOnce: boolean;
  Env: string[];
  Cmd?: string[];
  Image: string;
  Volumes?: Record<string, any>;
  WorkingDir: string;
  Entrypoint?: string[];
  OnBuild?: string[];
  Labels: Record<string, string>;
}

export interface ContainerStats {
  id: string;
  name: string;
  cpuPercent: number;
  memUsage: number;
  memLimit: number;
  memPercent: number;
  netInput: number;
  netOutput: number;
  blockInput: number;
  blockOutput: number;
  pidsCurrent: number;
  timestamp: string;
}

export interface DockerImage {
  Id: string;
  ParentId: string;
  RepoTags: string[];
  RepoDigests: string[];
  Created: number;
  Size: number;
  VirtualSize: number;
  SharedSize: number;
  Labels?: Record<string, string>;
  Containers: number;
}

export interface DockerVolume {
  Name: string;
  Driver: string;
  Mountpoint: string;
  CreatedAt?: string;
  Status?: Record<string, any>;
  Labels?: Record<string, string>;
  Scope: string;
  Options?: Record<string, string>;
  size?: number;
}

export interface DockerNetwork {
  Id: string;
  Name: string;
  Created: string;
  Scope: string;
  Driver: string;
  EnableIPv6: boolean;
  IPAM: {
    Driver: string;
    Options?: Record<string, string>;
    Config: Array<{
      Subnet?: string;
      Gateway?: string;
    }>;
  };
  Internal: boolean;
  Attachable: boolean;
  Ingress: boolean;
  ConfigFrom?: {
    Network: string;
  };
  ConfigOnly: boolean;
  Containers?: Record<string, any>;
  Options?: Record<string, string>;
  Labels?: Record<string, string>;
}

export interface DockerSystemInfo {
  ID: string;
  Containers: number;
  ContainersRunning: number;
  ContainersPaused: number;
  ContainersStopped: number;
  Images: number;
  Driver: string;
  DriverStatus: Array<[string, string]>;
  SystemStatus?: Array<[string, string]>;
  Plugins: {
    Volume: string[];
    Network: string[];
    Authorization?: string[];
    Log: string[];
  };
  MemoryLimit: boolean;
  SwapLimit: boolean;
  KernelMemory: boolean;
  KernelMemoryTCP: boolean;
  CpuCfsPeriod: boolean;
  CpuCfsQuota: boolean;
  CPUShares: boolean;
  CPUSet: boolean;
  PidsLimit: boolean;
  IPv4Forwarding: boolean;
  BridgeNfIptables: boolean;
  BridgeNfIp6tables: boolean;
  Debug: boolean;
  NFd: number;
  OomKillDisable: boolean;
  NGoroutines: number;
  SystemTime: string;
  LoggingDriver: string;
  CgroupDriver: string;
  CgroupVersion: string;
  NEventsListener: number;
  KernelVersion: string;
  OperatingSystem: string;
  OSVersion: string;
  OSType: string;
  Architecture: string;
  IndexServerAddress: string;
  RegistryConfig: any;
  NCPU: number;
  MemTotal: number;
  GenericResources?: any;
  DockerRootDir: string;
  HttpProxy: string;
  HttpsProxy: string;
  NoProxy: string;
  Name: string;
  Labels: string[];
  ExperimentalBuild: boolean;
  ServerVersion: string;
  ClusterStore?: string;
  ClusterAdvertise?: string;
  Runtimes?: Record<string, any>;
  DefaultRuntime: string;
  Swarm: any;
  LiveRestoreEnabled: boolean;
  Isolation: string;
  InitBinary: string;
  ContainerdCommit: {
    ID: string;
    Expected: string;
  };
  RuncCommit: {
    ID: string;
    Expected: string;
  };
  InitCommit: {
    ID: string;
    Expected: string;
  };
  SecurityOptions: string[];
  ProductLicense?: string;
  DefaultAddressPools?: any[];
  Warnings?: string[];
}

export interface DiskUsage {
  LayersSize: number;
  Images: DockerImage[];
  Containers: Container[];
  Volumes: DockerVolume[];
  BuildCache?: any[];
  BuilderSize?: number;
}

export interface SystemPruneReport {
  containersDeleted: string[];
  spaceReclaimed: number;
  imagesDeleted: string[];
  volumesDeleted: string[];
}
