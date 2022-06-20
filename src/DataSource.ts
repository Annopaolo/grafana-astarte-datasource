import type { DataSourceInstanceSettings, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';

import { AppEngineQuery, AppEngineDataSourceOptions } from './types';

export class DataSource extends DataSourceWithBackend<AppEngineQuery, AppEngineDataSourceOptions> {
  jsonData: AppEngineDataSourceOptions;

  constructor(instanceSettings: DataSourceInstanceSettings<AppEngineDataSourceOptions>) {
    super(instanceSettings);

    this.jsonData = instanceSettings.jsonData;
  }

  applyTemplateVariables(query: AppEngineQuery, scopedVars: ScopedVars): AppEngineQuery {
    const apply = (text: string) => getTemplateSrv().replace(text, scopedVars);

    return {
      ...query,
      device: apply(query.device),
      interfaceName: apply(query.interfaceName),
      path: apply(query.path),
    };
  }
}
