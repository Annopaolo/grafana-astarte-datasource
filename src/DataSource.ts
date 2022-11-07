import { DataSourceInstanceSettings } from '@grafana/data';
import { DataSourceWithBackend } from '@grafana/runtime';

import { AppEngineQuery, AppEngineDataSourceOptions } from './types';

export class DataSource extends DataSourceWithBackend<AppEngineQuery, AppEngineDataSourceOptions> {
  jsonData: AppEngineDataSourceOptions;

  constructor(instanceSettings: DataSourceInstanceSettings<AppEngineDataSourceOptions>) {
    super(instanceSettings);

    this.jsonData = instanceSettings.jsonData;
  }
}
