export interface CompanyOnboardingInput {
  company_name: string
  website: string
  support_needs: string[]
  files: File[]
}

export interface BuildPassportPayload {
  company_name: string
  website: string
  support_needs: string[]
  source_names: string[]
}

export function buildPassportPayload(input: CompanyOnboardingInput): BuildPassportPayload {
  const { files, ...company } = input
  return { ...company, source_names: files.map(file => file.name) }
}
