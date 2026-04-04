const BASE_URL = '/api';

export interface Service {
  id: string;
  name: string;
}

export interface GrpcView {
  serviceName: string;
  services: GrpcService[];
}

export interface GrpcService {
  name: string;
  methods: GrpcMethod[];
}

export interface GrpcMethod {
  name: string;
  input: GrpcMessage;
  output: GrpcMessage;
  consumers: string[];
}

export interface GrpcMessage {
  name: string;
  fields: GrpcField[];
}

export interface GrpcField {
  name: string;
  type: string;
  number: number;
  consumers: string[];
}

async function fetchJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`);
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}: ${res.statusText}`);
  }
  return res.json();
}

export function listServices(): Promise<{ services: Service[] }> {
  return fetchJSON('/services');
}

export function getGrpcView(serviceName: string): Promise<GrpcView> {
  return fetchJSON(`/services/${encodeURIComponent(serviceName)}/grpc-view`);
}
