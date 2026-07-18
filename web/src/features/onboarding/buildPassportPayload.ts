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
  source_ids: string[]
}

export function buildPassportPayload(input: CompanyOnboardingInput, sourceIds: string[]): BuildPassportPayload {
  const { files, ...company } = input
  return { ...company, source_names: files.map(file => file.name), source_ids: sourceIds }
}
