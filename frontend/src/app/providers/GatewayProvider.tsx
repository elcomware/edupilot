import { createContext, useContext, type ReactNode } from 'react'
import type { FinanceGateway } from '@/gateway'

const GatewayContext = createContext<FinanceGateway | null>(null)

export interface GatewayProviderProps {
  readonly gateway: FinanceGateway
  readonly children: ReactNode
}

export function GatewayProvider({ gateway, children }: GatewayProviderProps) {
  return <GatewayContext.Provider value={gateway}>{children}</GatewayContext.Provider>
}

export function useGateway(): FinanceGateway {
  const gateway = useContext(GatewayContext)
  if (gateway === null) {
    throw new Error('useGateway must be used inside a GatewayProvider')
  }
  return gateway
}
